package p2p

import (
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"net"

	"github.com/zeebo/blake3"
)

// HandleConnection 连接处理函数
func HandleConnection(conn net.Conn, mainState *models.MainStore, nodeId chan [32]byte, isInitiator bool) {
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
	c := Connection{
		Conn:         conn,
		WriteQueue:   make(chan []byte, 256),
		Done:         make(chan struct{}),
		Channels:     make(map[[32]byte]chan []byte),
		SessionState: int32(StateWaitingInitial),
		IsInitiator:  isInitiator,
		NodeId:       nodeId,
		Ready:        make(chan struct{}),
	}

	// 外部接口
	ioc := models.IOChannel{
		WriteQueue:   c.WriteQueue,
		Channels:     c.Channels,
		ChannelsLock: &c.ChannelsLock,
		Done:         c.Done,
		OnceDone:     &c.OnceDone,
		Ready:        c.Ready,
	}

	c.IOC = &ioc
	c.MainState = mainState
	c.h = handler

	go c.readLoop()
	go c.writeLoop()
	go c.KeepHeartbeat()

	logger.Debug("Connection opened: %v", conn.RemoteAddr())

	<-c.Done

	ct.Lock.Lock()
	delete(ct.Connection, blake3.Sum256(c.RemoteHandshake.PK[:]))
	ct.Lock.Unlock()

	return
}
