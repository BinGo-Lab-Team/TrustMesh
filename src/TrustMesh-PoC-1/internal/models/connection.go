package models

import "sync"

// IOChannel 外部读写接口
type IOChannel struct {
	WriteQueue   chan []byte
	Channels     map[[32]byte]chan []byte
	ChannelsLock *sync.RWMutex
	Done         chan struct{}
	OnceDone     *sync.Once
	Ready        chan struct{}
	NodeId       [32]byte
}

// ConnectionTable 连接表
type ConnectionTable struct {
	Lock       sync.RWMutex
	Connection map[[32]byte]*IOChannel
}

// makeConnectionTable 初始化连接表
func makeConnectionTable() *ConnectionTable {
	out := ConnectionTable{
		Connection: make(map[[32]byte]*IOChannel),
	}

	return &out
}
