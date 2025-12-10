package p2p

import (
	"TrustMesh-PoC-1/internal/db"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/table"
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeebo/blake3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NodeList 汇报的节点列表
type NodeList struct {
	Mu   sync.Mutex
	Node map[[32]byte]string
}

// BootstrapConnection 引导节点用连接结构体
type BootstrapConnection struct {
	Conn            net.Conn
	Done            chan struct{}
	Once            sync.Once
	SessionState    int32
	Ready           chan struct{}
	Node            *NodeList
	LocalHandshake  HandshakeMetadata
	RemoteHandshake HandshakeMetadata
}

func processingBootstrapReport(message []byte, node *NodeList, nodeId [32]byte, db *gorm.DB) {
	peer := table.Peer{
		NodeID:     nodeId[:],
		Address:    string(message),
		Reputation: 0,
		LastSeen:   uint64(time.Now().UnixMilli()),
		Status:     "FROM Report",
	}

	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_id"}},
		UpdateAll: true,
	}).Create(&peer)

	node.Mu.Lock()
	node.Node[nodeId] = string(message)
	node.Mu.Unlock()
}

func processingBootstrapReply(payload []byte, db *gorm.DB) {
	for i := 0; i < len(payload); {
		// 检查是否够解析 32B nodeId
		if i+32 > len(payload) {
			return
		}

		// 取 nodeId（值拷贝）
		var nodeId [32]byte
		copy(nodeId[:], payload[i:i+32])
		i += 32

		// 检查分隔符 '+'
		if i >= len(payload) || payload[i] != '+' {
			return
		}
		i++ // 跳过 '+'

		// 查找记录结束符 ';'
		semi := bytes.IndexByte(payload[i:], ';')
		if semi == -1 {
			return
		}

		// 拷贝 addr（深拷贝，防止 aliasing）
		addr := make([]byte, 0, semi)
		addr = append(addr, payload[i:i+semi]...)
		i += semi + 1 // 跳过 addr 和 ';'

		peer := table.Peer{
			NodeID:     nodeId[:],
			Address:    string(addr),
			Reputation: 0,
			LastSeen:   uint64(time.Now().UnixMilli()),
			Status:     "FROM Bootstrap",
		}

		db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_id"}},
			UpdateAll: true,
		}).Create(&peer)

		logger.Info("Bootstrap reply received, you can close this node")
	}
}

// bootstrapReadLoop 读通道协程
func (c *BootstrapConnection) bootstrapReadLoop() {
	defer c.Once.Do(func() { close(c.Done) })

	// 读循环
	for {
		select {
		case <-c.Done:
			return
		default:
			// 读取包头
			var headerBuf [4]byte
			if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if _, err := io.ReadFull(c.Conn, headerBuf[:]); err != nil {
				return
			}
			protocolId := binary.BigEndian.Uint32(headerBuf[:])

			// 握手状态
			state := atomic.LoadInt32(&c.SessionState)

			logger.Test("Read header: %v", protocolId)

			// 握手
			if state != int32(StateCompleted) {
				if state == int32(StateWaitingInitial) && protocolId == MsgHandshakeHello {
					var bodyBuf [72]byte
					// 读取 Hello 消息
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
						return
					}

					// 处理消息
					writeBuf, remote, local, isPass := processingHandshakeHello(bodyBuf)
					c.RemoteHandshake = remote
					c.LocalHandshake = local

					// 变更状态
					if isPass {
						if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
							return
						}
						if _, err := c.Conn.Write(writeBuf[:]); err != nil {
							return
						}
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingInitial), int32(StateWaitingReply)); !success {
							return
						}
						continue
					} else {
						return
					}
				} else if state == int32(StateWaitingReply) && protocolId == MsgHandshakeConfirm {
					var bodyBuf [64]byte
					// 读取 Confirm 消息
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
						return
					}

					// 处理消息
					isPass := processingHandshakeConfirm(bodyBuf, c.RemoteHandshake, c.LocalHandshake)

					// 变更状态
					if isPass {
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingReply), int32(StateCompleted)); !success {
							return
						}
						nodeId := blake3.Sum256(c.RemoteHandshake.PK[:])

						// 通知握手完成
						close(c.Ready)

						logger.Debug("Handshake done: %v", nodeId)

						continue
					} else {
						return
					}
				} else {
					return
				}
			}

			// 业务路由
			switch protocolId {
			case MsgHeartbeat:
				var bodyBuf [4]byte
				if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return
				}
				if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
					return
				}
			case MsgBootstrapReport:
				bodyBuf, err := lenReadUnit32(c.Conn, math.MaxInt32, 10*time.Second)
				if err != nil {
					return
				}
				go processingBootstrapReport(bodyBuf, c.Node, blake3.Sum256(c.RemoteHandshake.PK[:]), db.GetDB())
				return
			default:
				return
			}
		}
	}
}

// BootstrapHandleConnection 连接管理函数
func BootstrapHandleConnection(conn net.Conn, resp *NodeList, mainState *models.MainStore) {
	ct := mainState.ConnectionTable

	// 退出时关闭连接
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Warning("Close Connection Error: %v", err)
		}
		logger.Debug("Close Connection: %v", conn.RemoteAddr())
	}(conn)

	// 连接信息
	c := BootstrapConnection{
		Conn:         conn,
		Done:         make(chan struct{}),
		SessionState: int32(StateWaitingInitial),
		Ready:        make(chan struct{}),
		Node:         resp,
	}

	go c.bootstrapReadLoop()

	logger.Debug("Bootstrap Connection opened: %v", conn.RemoteAddr())

	<-c.Done

	ct.Lock.Lock()
	delete(ct.Connection, blake3.Sum256(c.RemoteHandshake.PK[:]))
	ct.Lock.Unlock()

	return
}
