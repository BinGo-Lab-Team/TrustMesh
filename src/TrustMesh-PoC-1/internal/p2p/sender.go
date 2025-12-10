package p2p

import (
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/table"
	"TrustMesh-PoC-1/internal/tools"
	"net"
	"time"

	"gorm.io/gorm"
)

// SendNodeIdMessage 使用已知节点的 NodeId 发送消息
func SendNodeIdMessage(nodeId [32]byte, message []byte, db *gorm.DB, mainState *models.MainStore) bool {
	ct := mainState.ConnectionTable

	// 尝试获取 IO通道
	ct.Lock.RLock()
	ioChan, exists := ct.Connection[nodeId]
	ct.Lock.RUnlock()

	if exists {
		select {
		case <-ioChan.Done:
			logger.Warning("Connection not deleted from ConnectionTable, BUT connection is closed!")
		default:
			select {
			case ioChan.WriteQueue <- message:
				return true
			default:
				return false
			}
		}
	}

	// 根据节点 ID 查询对方地址
	var nid = nodeId[:]
	var addr string
	err := db.Model(&table.Peer{}).
		Select("address").
		Where("node_id = ?", nid).
		Take(&addr).Error
	if err != nil {
		logger.Error(err.Error())
	}

	// 检查地址是否有效
	if !tools.IsValidForDial(addr) {
		logger.Warning("The address [%v] is not valid for this node", addr)
		return false
	}

	// 连接
	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Error(err.Error())
		return false
	}

	ready := make(chan [32]byte, 1)
	done := make(chan struct{})
	go HandleConnection(rawConn, mainState, ready, true)

	select {
	case realID := <-ready:
		close(done)

		// 检查对方 ID 和指定 ID 是否吻合
		if realID != nodeId {
			return false
		}

		ct.Lock.RLock()
		newConn, ok := ct.Connection[realID]
		ct.Lock.RUnlock()

		if !ok {
			return false
		}

		select {
		case newConn.WriteQueue <- message:
			return true
		default:
			return false
		}
	case <-tools.WaitTimeout(done, 3*time.Second):
		return false
	}
}
