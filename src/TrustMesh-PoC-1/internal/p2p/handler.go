package p2p

import "TrustMesh-PoC-1/internal/models"

var (
	handler Handler
)

// Init 初始化接口
func Init(h Handler) {
	handler = h
}

// Handler 顶层接口
type Handler interface {
	Consensus
}

// Consensus 共识处理接口
type Consensus interface {
	ProcessingInquiry(b [72]byte, mainState *models.MainStore, ioc *models.IOChannel)
	ProcessingInquiryReply(b [36]byte, ioc *models.IOChannel)
	ProcessingProposalBody(b []byte, mainState *models.MainStore)
	ProcessProposalSig(b []byte, mainState *models.MainStore)
}
