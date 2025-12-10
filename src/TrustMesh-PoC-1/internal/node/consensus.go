package node

import (
	"TrustMesh-PoC-1/internal/consensus"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/p2p"
	"TrustMesh-PoC-1/internal/tools"
	"crypto/ed25519"
	"encoding/binary"
	"time"

	"github.com/zeebo/blake3"
)

// Node 实现结构体
type Node struct{}

// ProcessingInquiry 处理问询信息
func (Node) ProcessingInquiry(b [72]byte, mainState *models.MainStore, ioc *models.IOChannel) {
	proposalState := mainState.ProposalSate

	// 轮次
	var round int64
	round = int64(binary.BigEndian.Uint64(b[0:8]))
	// 提案哈希
	var pHash [32]byte
	copy(pHash[:], b[8:40])
	// 事务 ID
	var transactionId [32]byte
	copy(transactionId[:], b[40:72])

	proposalState.UpdateLock.RLock()
	_, exists := proposalState.Update[round]
	proposalState.UpdateLock.RUnlock()

	message := make([]byte, 0, 4+32+4)

	// header
	var header [4]byte
	binary.BigEndian.PutUint32(header[0:4], p2p.MsgInquiryReply)
	message = append(message, header[:]...)
	// 事务 ID
	message = append(message, transactionId[:]...)

	var payload [4]byte
	if exists {
		proposalState.DataLock.RLock()
		_, exists = proposalState.Data[round][pHash]
		proposalState.DataLock.RUnlock()
		if exists {
			binary.BigEndian.PutUint32(payload[:], p2p.TrueOrYes)
		} else {
			binary.BigEndian.PutUint32(payload[:], p2p.FalseOrNo)
		}
	} else {
		logger.Test("The round [%v] doesn't exist", round)
		binary.BigEndian.PutUint32(payload[:], p2p.RefuseOrNoNeed)
	}
	// payload
	message = append(message, payload[:]...)

	select {
	case <-ioc.Done:
		return
	default:
		ok := make(chan struct{})
		select {
		case ioc.WriteQueue <- message:
			return
		case <-tools.WaitTimeout(ok, 3*time.Second):
			return
		}
	}
}

// ProcessingInquiryReply 处理问询回复
func (Node) ProcessingInquiryReply(b [36]byte, ioc *models.IOChannel) {
	var transactionId [32]byte
	copy(transactionId[:], b[0:32])

	var payload [4]byte
	copy(payload[:], b[32:36])

	ioc.ChannelsLock.RLock()
	ch, exists := ioc.Channels[transactionId]
	if exists {
		select {
		case ch <- payload[:]:
		default:
		}
	} else {
		logger.Test("Failed to find channel")
	}
	ioc.ChannelsLock.RUnlock()
}

// ProcessingProposalBody 处理提案本体
func (Node) ProcessingProposalBody(b []byte, mainState *models.MainStore) {
	if len(b) <= 8+32+8+64 {
		logger.Debug("Proposal body length failed")
		return
	}

	var proposal models.ProposalBody

	// 轮次
	var round int64
	round = int64(binary.BigEndian.Uint64(b[0:8]))
	// 公钥
	var pk [32]byte
	copy(pk[:], b[8:40])
	proposal.ProposerPubKey = pk
	// 时间戳
	proposal.Timestamp = binary.BigEndian.Uint64(b[40:48])
	// 签名
	var sig [64]byte
	copy(sig[:], b[48:112])
	proposal.ProposerSig = sig
	// 载荷
	proposal.Payload = b[112:]
	// NodeId
	nodeId := blake3.Sum256(pk[:])
	// 标签
	var PUBLISHV1DomainTag [4]byte
	binary.BigEndian.PutUint32(PUBLISHV1DomainTag[:], consensus.PROPOSERV1Domain)

	// 计算 pHash
	pHashData := make([]byte, 0, 8+32+8+len(proposal.Payload))
	pHashData = append(pHashData, b[0:8]...)
	pHashData = append(pHashData, nodeId[:]...)
	pHashData = append(pHashData, b[40:48]...)
	pHashData = append(pHashData, proposal.Payload...)
	pHash := blake3.Sum256(pHashData)

	// 计算签名数据
	sigData := make([]byte, 0, 4+8+32)
	sigData = append(sigData, PUBLISHV1DomainTag[:]...)
	sigData = append(sigData, b[0:8]...)
	sigData = append(sigData, pHash[:]...)
	sigDataHash := blake3.Sum256(sigData)

	if !ed25519.Verify(pk[:], sigDataHash[:], sig[:]) {
		logger.Debug("Proposal Body %v Sig verification failed, pk: %v, sigData: %v", pHash, pk, sigData)
		return
	}

	proposalState := mainState.ProposalSate
	proposalState.DataLock.Lock()
	_, ok := proposalState.Data[round]
	_, e := proposalState.Data[round][pHash]
	if !ok || e {
		proposalState.DataLock.Unlock()
		return
	}
	proposalState.Data[round][pHash] = proposal
	proposalState.DataLock.Unlock()
}

// ProcessProposalSig 处理提案签名集
func (Node) ProcessProposalSig(b []byte, mainState *models.MainStore) {
	if len(b) <= 8+32+2 {
		return
	}

	var round int64
	round = int64(binary.BigEndian.Uint64(b[0:8]))

	var pHash [32]byte
	copy(pHash[:], b[8:40])

	var signerCount int
	signerCount = int(binary.BigEndian.Uint16(b[40:42]))

	// TODO: 剪枝
	idx := 42
	for i := 0; i < signerCount; i++ {
		if len(b) < idx+32+4+8+64+2 {
			logger.Debug("ProcessProposalSig too short")
			continue
		}

		var sig models.Attestation

		// 公钥
		copy(sig.SignerPubKey[:], b[idx:idx+32])
		idx += 32
		// 分数
		scoreByte := b[idx : idx+4]
		sig.Score = binary.BigEndian.Uint32(b[idx : idx+4])
		idx += 4
		// 时间戳
		timestampByte := b[idx : idx+8]
		sig.Timestamp = binary.BigEndian.Uint64(b[idx : idx+8])
		idx += 8
		// 签名
		copy(sig.Signature[:], b[idx:idx+64])
		idx += 64
		// 担保数量
		var guaranteeCount int
		guaranteeCount = int(binary.BigEndian.Uint16(b[idx : idx+2]))
		idx += 2
		// nodeId
		nodeId := blake3.Sum256(sig.SignerPubKey[:])

		// Tag
		var RATEV1DomainTag [4]byte
		binary.BigEndian.PutUint32(RATEV1DomainTag[:], consensus.RATEV1Domain)

		sigData := make([]byte, 0, 4+8+32+4+8)
		sigData = append(sigData, RATEV1DomainTag[:]...)
		sigData = append(sigData, b[0:8]...)
		sigData = append(sigData, pHash[:]...)
		sigData = append(sigData, scoreByte[:]...)
		sigData = append(sigData, timestampByte[:]...)
		sigDataHash := blake3.Sum256(sigData)

		if !ed25519.Verify(sig.SignerPubKey[:], sigDataHash[:], sig.Signature[:]) {
			logger.Debug("ProcessProposalSig %v Sig verification failed", sig)
			continue
		}

		// 将分数写入提案
		mainState.ProposalSate.SigLock.Lock()
		_, ok := mainState.ProposalSate.Sig[round]
		_, existsP := mainState.ProposalSate.Sig[round][pHash]

		if !ok {
			mainState.ProposalSate.SigLock.Unlock()
			return
		}

		if !existsP {
			mainState.ProposalSate.Sig[round][pHash] = make(map[[32]byte]models.Attestation)
		} else {
			val, existsS := mainState.ProposalSate.Sig[round][pHash][nodeId]
			if existsS {
				if val.Score >= sig.Score {
					mainState.ProposalSate.SigLock.Unlock()
					continue
				} else {
					logger.Test("Score [%v] to [%v] & pHash: %v, nodeId: %v", val.Score, sig.Score, pHash, nodeId)
				}
			}
		}

		mainState.ProposalSate.Sig[round][pHash][nodeId] = sig
		mainState.ProposalSate.SigLock.Unlock()

		for j := 0; j < guaranteeCount; j++ {
			var guaranteeNodeId [32]byte
			copy(guaranteeNodeId[:], b[idx:idx+32])
			idx += 32

			var guaranteeSig [64]byte
			copy(guaranteeSig[:], b[idx:idx+64])
			idx += 64

			// TODO: 验证担保并写入
		}
	}

	// 通知处理
	mainState.ProposalSate.UpdateLock.RLock()
	ch, exists := mainState.ProposalSate.Update[round]
	if exists {
		select {
		case ch <- pHash:
			//logger.Debug("ProcessProposalSig channel ok")
		default:
			//logger.Debug("ProcessProposalSig channel send failed")
		}
	}
	mainState.ProposalSate.UpdateLock.RUnlock()
}
