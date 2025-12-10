package consensus

import (
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/p2p"
	"TrustMesh-PoC-1/internal/table"
	"TrustMesh-PoC-1/internal/tools"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"gorm.io/gorm"
)

// sendProposal 发送提案，不允许异步执行
func sendProposal(mainState *models.MainStore, round int64, proposalHash [32]byte, db *gorm.DB, hop int) error {
	proposalState := mainState.ProposalSate
	ct := mainState.ConnectionTable

	// 构建问询消息
	message := make([]byte, 0, 4+8+32)

	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], p2p.MsgInquiryHaveProposal)
	message = append(message, header[:]...)
	// 写入询问的轮次
	var roundBytes [8]byte
	binary.BigEndian.PutUint64(roundBytes[:], uint64(round))
	message = append(message, roundBytes[:]...)
	// 写入询问的提案哈希
	message = append(message, proposalHash[:]...)

	peers := make([]table.Peer, 0)
	err := db.
		Select("node_id").
		Order("RANDOM()").
		Limit(hop).
		Find(&peers).Error
	if err != nil {
		return fmt.Errorf("failed to find peers: %w", err)
	}

	proposalState.DataLock.RLock()
	if _, e := proposalState.Data[round][proposalHash]; !e {
		logger.Error("failed to find proposal state: %v", proposalHash)
	}
	proposalData := cloneProposalBody(proposalState.Data[round][proposalHash])
	proposalState.DataLock.RUnlock()

	proposalState.SigLock.RLock()
	att := cloneAttestationMap(proposalState.Sig[round][proposalHash])
	proposalState.SigLock.RUnlock()

	proposalState.GuaranteeLock.RLock()
	gua := cloneGuaranteeMapMap(proposalState.Guarantee[round][proposalHash])
	proposalState.GuaranteeLock.RUnlock()

	proposalBodyMsg := buildProposalBodyMessage(proposalData, round)
	proposalSigMsg := buildProposalSigMessage(att, gua, round, proposalHash)

	// 循环发送
	for _, peer := range peers {
		go func() {
			if len(peer.NodeID) != 32 {
				logger.Error("The length of NodeId in database is incorrect! Actual length: %v Problems line: %v", len(peer.NodeID), peer.NodeID)
				return
			}
			var nodeId [32]byte
			copy(nodeId[:], peer.NodeID)

			ct.Lock.RLock()
			ioChan, exists := ct.Connection[nodeId]
			ct.Lock.RUnlock()

			// 如果没有已有连接就创建
			if !exists {
				// 根据节点 ID 查询对方地址
				var addr string
				err := db.Model(&table.Peer{}).
					Select("address").
					Where("node_id = ?", peer.NodeID).
					Take(&addr).Error
				if err != nil {
					logger.Error(err.Error())
					return
				}

				// 连接
				rawConn, err := net.Dial("tcp", addr)
				if err != nil {
					logger.Debug(err.Error())
					return
				}

				// 将连接移交给管理函数
				ready := make(chan [32]byte, 1)
				go p2p.HandleConnection(rawConn, mainState, ready, true)

				// 等待连接完毕
				done := make(chan struct{})
				select {
				case realID := <-ready:
					close(done)

					// 检查对方 ID 和指定 ID 是否吻合
					if realID != nodeId {
						logger.Debug("The node id is incorrect: %v", realID)
						return
					}

					// 重新赋值
					ct.Lock.RLock()
					ioChan, exists = ct.Connection[nodeId]
					ct.Lock.RUnlock()

					if !exists {
						logger.Debug("The connection is ready but doesn't exist")
						return
					}
				case <-tools.WaitTimeout(done, 5*time.Second):
					return
				}
			}

			// 发送消息
			select {
			case <-ioChan.Done:
				logger.Warning("Connection not deleted from ConnectionTable, BUT connection is closed!")
				return
			case <-ioChan.Ready:
				// 生成事务 ID
				var transaction [32]byte
				if _, err := rand.Read(transaction[:]); err != nil {
					logger.Error("Get nonce failed: %v", err)
					return
				}

				sendMessage := message
				sendMessage = append(sendMessage, transaction[:]...)

				// 注册清理逻辑
				defer func() {
					ioChan.ChannelsLock.Lock()
					delete(ioChan.Channels, transaction)
					ioChan.ChannelsLock.Unlock()
				}()

				// 写入返回通道
				isHave := make(chan []byte, 1)
				ioChan.ChannelsLock.Lock()
				ioChan.Channels[transaction] = isHave
				ioChan.ChannelsLock.Unlock()

				// 发送消息
				{
					ok := make(chan struct{})
					select {
					case ioChan.WriteQueue <- sendMessage:
						close(ok)
					case <-tools.WaitTimeout(ok, 3*time.Second):
						logger.Debug("Send 'proposal is have?' timeout!")
						return
					}
				}

				// 等待返回消息
				ok := make(chan struct{})
				select {
				case b := <-isHave:
					close(ok)

					// 检查返回长度防止 panic
					if len(b) < 4 {
						logger.Debug("The length of isHave is incorrect! Actual length: %v", len(b))
						return
					}

					// 发送提案本体
					reply := binary.BigEndian.Uint32(b)
					if reply == p2p.TrueOrYes {
						logger.Test("remote node already have")
					} else if reply == p2p.RefuseOrNoNeed {
						logger.Test("remote node refuse")
						return
					} else if reply == p2p.FalseOrNo {
						logger.Test("remote node need")
						done := make(chan struct{})
						select {
						case ioChan.WriteQueue <- proposalBodyMsg:
							close(done)
						case <-tools.WaitTimeout(done, 3*time.Second):
							return
						}
					} else {
						logger.Debug("The field of isHave is incorrect")
						return
					}

					// 发送签名集
					done := make(chan struct{})
					select {
					case ioChan.WriteQueue <- proposalSigMsg:
						close(done)
					case <-tools.WaitTimeout(done, 5*time.Second):
						return
					}
				case <-tools.WaitTimeout(ok, 10*time.Second):
					logger.Debug("Wait 'proposal is have?' reply timeout!")
					return
				}
			}
		}()
	}

	return nil
}
