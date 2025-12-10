package models

import "sync"

// ProposalBody 提案数据
type ProposalBody struct {
	ProposerPubKey [32]byte
	Payload        []byte
	Timestamp      uint64
	ProposerSig    [64]byte
}

// Attestation 签名打分信息
type Attestation struct {
	SignerPubKey [32]byte
	Score        uint32
	Timestamp    uint64
	Signature    [64]byte
}

// Guarantee 担保信息
type Guarantee struct {
	Signature [64]byte
}

// Score 本地分数信息
type Score struct {
	Score     uint32
	LastScore uint32
}

// ProposalStore 提案汇总结构体
type ProposalStore struct {
	// 轮次 | 提案哈希 | 提案内容
	Data     map[int64]map[[32]byte]ProposalBody
	DataLock sync.RWMutex
	// 轮次 | 提案哈希 | 签名者NodeId | 签名信息
	Sig     map[int64]map[[32]byte]map[[32]byte]Attestation
	SigLock sync.RWMutex
	// 轮次 | 提案哈希 | 担保者 | 被担保者 | 签名信息
	Guarantee     map[int64]map[[32]byte]map[[32]byte]map[[32]byte]Guarantee
	GuaranteeLock sync.RWMutex
	// 轮次 | 提案哈希 | 本地分数
	Score     map[int64]map[[32]byte]Score
	ScoreLock sync.RWMutex
	// 轮次 | 通知通道
	Update     map[int64]chan [32]byte
	UpdateLock sync.RWMutex
}

// makeProposalStore 初始化提案结构体
func makeProposalStore() *ProposalStore {
	out := ProposalStore{
		Data:      make(map[int64]map[[32]byte]ProposalBody),
		Sig:       make(map[int64]map[[32]byte]map[[32]byte]Attestation),
		Guarantee: make(map[int64]map[[32]byte]map[[32]byte]map[[32]byte]Guarantee),
		Score:     make(map[int64]map[[32]byte]Score),
		Update:    make(map[int64]chan [32]byte),
	}

	return &out
}
