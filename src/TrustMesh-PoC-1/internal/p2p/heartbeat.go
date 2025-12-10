package p2p

import (
	"TrustMesh-PoC-1/internal/keys"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/tools"
	"encoding/binary"
	"time"
)

// sendHeartbeat 发送心跳包
func (c *Connection) sendHeartbeat() bool {
	privateKey, _, err := keys.LoadOrCreateKey()
	if err != nil {
		logger.Error("Failed to load or create key: %v", err)
		return false
	}
	var message []byte

	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MsgHeartbeat)
	message = append(message, header[:]...)

	// 创建 [4]byte 格式的 Tag
	var TMHBDomainTag [4]byte
	binary.BigEndian.PutUint32(TMHBDomainTag[:], TMHBDomain)
	message = append(message, TMHBDomainTag[:]...)

	// 发送到写入队列
	select {
	case c.WriteQueue <- message:
		keys.Zeroize(privateKey)
		return true
	default:
		keys.Zeroize(privateKey)
		return false
	}
}

// KeepHeartbeat 定时发送心跳包
func (c *Connection) KeepHeartbeat() {
	for {
		select {
		case <-c.Done:
			return
		case <-time.After(5 * time.Second):
			ok := make(chan struct{})
			select {
			case <-c.Ready:
				close(ok)
				c.sendHeartbeat()
			case <-tools.WaitTimeout(ok, 5*time.Second):
				c.OnceDone.Do(func() { close(c.Done) })
				return
			}
		}
	}
}

func processingHeartbeat(message [4]byte) bool {
	// 创建 [4]byte 格式的 Tag
	var TMHBDomainTag [4]byte
	binary.BigEndian.PutUint32(TMHBDomainTag[:], TMHBDomain)

	if TMHBDomainTag == message {
		return true
	} else {
		return false
	}
}
