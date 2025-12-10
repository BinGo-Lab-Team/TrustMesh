package consensus

import (
	"TrustMesh-PoC-1/internal/keys"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/p2p"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
)

// buildProposalBodyMessage 构建提案数据消息
func buildProposalBodyMessage(store models.ProposalBody, round int64) []byte {
	message := make([]byte, 0, 4+4+(8+32+8+64+len(store.Payload)))

	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], p2p.MsgProposalBody)
	message = append(message, header[:]...)

	// 轮次
	var roundBytes [8]byte
	binary.BigEndian.PutUint64(roundBytes[:], uint64(round))
	// 提案时间戳
	var timestamp [8]byte
	binary.BigEndian.PutUint64(timestamp[:], store.Timestamp)

	// 提案内容
	msgPayload := make([]byte, 0, 8+32+8+64+len(store.Payload))
	msgPayload = append(msgPayload, roundBytes[:]...)
	msgPayload = append(msgPayload, store.ProposerPubKey[:]...)
	msgPayload = append(msgPayload, timestamp[:]...)
	msgPayload = append(msgPayload, store.ProposerSig[:]...)
	msgPayload = append(msgPayload, store.Payload...)

	// 写入提案长度
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(msgPayload))) // 溢出风险
	message = append(message, length[:]...)

	// 写入提案数据
	message = append(message, msgPayload...)

	return message
}

// buildProposalSigMessage 构建提案签名&担保消息
func buildProposalSigMessage(sigList map[[32]byte]models.Attestation, guarantee map[[32]byte]map[[32]byte]models.Guarantee, round int64, proposalHash [32]byte) []byte {
	// (没有计算被担保者签名)
	message := make([]byte, 0, 4+4+8+32+2+((32+4+8+64+2)*len(sigList)))

	// header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], p2p.MsgProposalSig)
	message = append(message, header[:]...)

	// payload
	payload := make([]byte, 0, 8+32+2+((32+4+8+64+2)*len(sigList)))
	// 写入轮次
	var roundByte [8]byte
	binary.BigEndian.PutUint64(roundByte[:], uint64(round))
	payload = append(payload, roundByte[:]...)
	// 写入提案哈希
	payload = append(payload, proposalHash[:]...)
	// 写入签名者数量
	var signerCount [2]byte
	binary.BigEndian.PutUint16(signerCount[:], uint16(len(sigList)))
	payload = append(payload, signerCount[:]...)

	for signerId, val := range sigList {
		// 公钥
		payload = append(payload, val.SignerPubKey[:]...)
		// 分数
		var score [4]byte
		binary.BigEndian.PutUint32(score[:], val.Score)
		payload = append(payload, score[:]...)
		// 时间戳
		var timestamp [8]byte
		binary.BigEndian.PutUint64(timestamp[:], val.Timestamp)
		payload = append(payload, timestamp[:]...)
		// 签名
		payload = append(payload, val.Signature[:]...)
		// 被担保者数量
		var count [2]byte
		binary.BigEndian.PutUint16(count[:], uint16(len(guarantee[signerId])))
		payload = append(payload, count[:]...)

		for k, v := range guarantee[signerId] {
			// 被担保者 NodeId
			payload = append(payload, k[:]...)
			// 签名
			payload = append(payload, v.Signature[:]...)
		}
	}

	// 长度
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(payload)))
	message = append(message, length[:]...)
	// 内容
	message = append(message, payload...)

	return message
}

// buildRateSig 构建打分签名
func buildRateSig(round int64, pHash [32]byte, score uint32) ([32]byte, models.Attestation, error) {
	priKey, pubKey, err := keys.LoadOrCreateKey()
	if err != nil {
		return [32]byte{}, models.Attestation{}, fmt.Errorf("failed to get keys: %w", err)
	}

	// 时间戳
	var timestampByte [8]byte
	timestamp := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(timestampByte[:], timestamp)
	// 公钥
	var pk [32]byte
	copy(pk[:], pubKey)
	// Tag
	var RATEV1DomainTag [4]byte
	binary.BigEndian.PutUint32(RATEV1DomainTag[:], RATEV1Domain)
	// Round
	var roundByte [8]byte
	binary.BigEndian.PutUint64(roundByte[:], uint64(round))
	// Score
	var scoreByte [4]byte
	binary.BigEndian.PutUint32(scoreByte[:], score)

	sigData := make([]byte, 0, 4+8+32+4+8)
	sigData = append(sigData, RATEV1DomainTag[:]...)
	sigData = append(sigData, roundByte[:]...)
	sigData = append(sigData, pHash[:]...)
	sigData = append(sigData, scoreByte[:]...)
	sigData = append(sigData, timestampByte[:]...)
	sigDataHash := blake3.Sum256(sigData)

	// 签名
	signature := ed25519.Sign(priKey, sigDataHash[:])
	keys.Zeroize(priKey)

	var sig [64]byte
	copy(sig[:], signature)

	out := models.Attestation{
		SignerPubKey: pk,
		Score:        score,
		Timestamp:    timestamp,
		Signature:    sig,
	}

	return blake3.Sum256(pubKey), out, nil
}
