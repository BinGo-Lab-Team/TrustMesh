package p2p

import (
	"TrustMesh-PoC-1/internal/models"
	"net"
	"sync"
)

// HandshakeState 握手状态
type HandshakeState int32

// HandshakeMetadata 握手信息
type HandshakeMetadata struct {
	PK    [32]byte
	Nonce [32]byte
	Time  uint64
}

// 标签
const (
	// TMHSV1Domain 握手 Challenge 标签
	TMHSV1Domain uint32 = 0x7D4B2507
	// TMHBDomain 心跳包
	TMHBDomain uint32 = 0xB02F55D8
)

// 表示字段
const (
	TrueOrYes      uint32 = 0x0D7EF654
	FalseOrNo      uint32 = 0xB059C403
	RefuseOrNoNeed uint32 = 0x62DBC647
)

// 握手状态机
const (
	StateWaitingInitial HandshakeState = iota // 待发送 or 待接收
	StateWaitingReply                         // 对方应答后，我等待回复
	StateCompleted                            // 握手成功
)

// 协议 ID
const (
	// MsgHandshakeHello 握手 Hello 消息
	MsgHandshakeHello uint32 = 0x00000001
	// MsgHandshakeResponse 握手 Response 消息
	MsgHandshakeResponse uint32 = 0x00000002
	// MsgHandshakeConfirm 握手 Confirm 消息
	MsgHandshakeConfirm uint32 = 0x00000003
	// MsgHeartbeat 心跳包
	MsgHeartbeat uint32 = 0x00000006
	// MsgBootstrapReport 汇报引导节点
	MsgBootstrapReport uint32 = 0x00000007
	// MsgBootstrapReply 引导节点回复
	MsgBootstrapReply uint32 = 0x00000008
	// MsgProposalBody 提案本体
	MsgProposalBody uint32 = 0x00000009
	// MsgProposalSig 提案签名集
	MsgProposalSig uint32 = 0x000000A
	// MsgInquiryHaveProposal 询问是否持有提案
	MsgInquiryHaveProposal uint32 = 0x0000000B
	// MsgInquiryReply 回复询问
	MsgInquiryReply uint32 = 0x0000000C
)

// Connection 内部接口
type Connection struct {
	// 连接对象
	Conn net.Conn
	// 写队列
	WriteQueue chan []byte
	// 回拨通道
	Channels map[[32]byte]chan []byte
	// 回拨通道互斥锁
	ChannelsLock sync.RWMutex
	// 结束信号
	Done chan struct{}
	// 结束信号 Once 对象
	OnceDone sync.Once
	// 握手状态
	SessionState int32
	// 是否为发起者
	IsInitiator bool
	// 程序主对象
	MainState *models.MainStore
	// 读写通道
	IOC *models.IOChannel
	// 准备就绪信号
	Ready chan struct{}
	// NodeId 汇报通道
	NodeId chan [32]byte
	// 接口
	h Handler
	// 本地握手信息
	LocalHandshake HandshakeMetadata
	// 对方握手信息
	RemoteHandshake HandshakeMetadata
}
