package p2p

import (
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
)

// ConnectionAlreadyExists 冗余连接处理函数
func ConnectionAlreadyExists(nodeId [32]byte, c *models.IOChannel) {
	logger.Debug("node %v already connected", nodeId)
	//c.OnceDone.Do(func() { close(c.Done) })
	return
}
