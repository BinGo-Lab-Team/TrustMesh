package consensus

import (
	"TrustMesh-PoC-1/internal/db"
	"TrustMesh-PoC-1/internal/keys"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
)

// ExecuteRound 轮次执行函数
func ExecuteRound(mainState *models.MainStore, round int64, interval time.Duration) error {
	proposalSate := mainState.ProposalSate

	// 删除轮次数据
	defer func() {
		proposalSate.UpdateLock.Lock()
		delete(proposalSate.Update, round)
		proposalSate.UpdateLock.Unlock()

		proposalSate.DataLock.Lock()
		delete(proposalSate.Data, round)
		proposalSate.DataLock.Unlock()

		proposalSate.SigLock.Lock()
		delete(proposalSate.Sig, round)
		proposalSate.SigLock.Unlock()

		proposalSate.GuaranteeLock.Lock()
		delete(proposalSate.Guarantee, round)
		proposalSate.GuaranteeLock.Unlock()

		proposalSate.ScoreLock.Lock()
		delete(proposalSate.Score, round)
		proposalSate.ScoreLock.Unlock()
	}()

	proposalSate.DataLock.Lock()
	proposalSate.Data[round] = make(map[[32]byte]models.ProposalBody)
	proposalSate.DataLock.Unlock()

	proposalSate.SigLock.Lock()
	proposalSate.Sig[round] = make(map[[32]byte]map[[32]byte]models.Attestation)
	proposalSate.SigLock.Unlock()

	proposalSate.GuaranteeLock.Lock()
	proposalSate.Guarantee[round] = make(map[[32]byte]map[[32]byte]map[[32]byte]models.Guarantee)
	proposalSate.GuaranteeLock.Unlock()

	proposalSate.ScoreLock.Lock()
	proposalSate.Score[round] = make(map[[32]byte]models.Score)
	proposalSate.ScoreLock.Unlock()

	// 注册处理通道
	pChan := make(chan [32]byte, 128)
	proposalSate.UpdateLock.Lock()
	proposalSate.Update[round] = pChan
	proposalSate.UpdateLock.Unlock()

	<-TimeNextRound(interval, round)

	// 随机信誉
	if err := RandomDBReputation(db.GetDB()); err != nil {
		logger.Error("Error in random db reputation")
	}

	// 获取密钥
	priKey, pubKey, err := keys.LoadOrCreateKey()
	if err != nil {
		return fmt.Errorf("failed to load or generate private key: %v", err)
	}
	// 生成载荷
	wordPass, err := WordPass(32)
	if err != nil {
		logger.Error("Error in generate WordPass")
	}

	// NodeId
	myNodeId := blake3.Sum256(pubKey)
	// 提案载荷
	myPayload := make([]byte, 0, len(wordPass))
	myPayload = append(myPayload[:], wordPass...)
	// 轮次
	var proposalRound [8]byte
	binary.BigEndian.PutUint64(proposalRound[:], uint64(round))
	// 提案时间戳
	var timestamp [8]byte
	proposalTime := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(timestamp[:], proposalTime)
	// 标签
	var PUBLISHV1DomainTag [4]byte
	binary.BigEndian.PutUint32(PUBLISHV1DomainTag[:], PROPOSERV1Domain)

	// 构建 pHash
	myProposalData := make([]byte, 0, 8+32+8+len(myPayload))
	myProposalData = append(myProposalData, proposalRound[:]...)
	myProposalData = append(myProposalData, myNodeId[:]...)
	myProposalData = append(myProposalData, timestamp[:]...)
	myProposalData = append(myProposalData, myPayload...)
	mypHash := blake3.Sum256(myProposalData)

	// 构建发起者签名
	initiatorData := make([]byte, 0, 4+8+32)
	initiatorData = append(initiatorData, PUBLISHV1DomainTag[:]...)
	initiatorData = append(initiatorData, proposalRound[:]...)
	initiatorData = append(initiatorData, mypHash[:]...)
	initiatorDataHash := blake3.Sum256(initiatorData)
	var initiatorSig [64]byte
	copy(initiatorSig[:], ed25519.Sign(priKey, initiatorDataHash[:]))
	keys.Zeroize(priKey)

	logger.Debug("makeProposal: %v", mypHash)

	// 构造提案结构体
	myProposal := models.ProposalBody{
		ProposerPubKey: [32]byte(pubKey),
		Payload:        myPayload,
		Timestamp:      proposalTime,
		ProposerSig:    initiatorSig,
	}

	// 构造打分结构体
	_, myAttestation, err := buildRateSig(round, mypHash, MyScore)
	if err != nil {
		logger.Error("Failed in build rate sig: %v", err)
	}

	// 写入提案
	proposalSate.DataLock.Lock()
	proposalSate.Data[round][mypHash] = myProposal
	proposalSate.DataLock.Unlock()

	// 写入打分签名
	proposalSate.SigLock.Lock()
	proposalSate.Sig[round][mypHash] = make(map[[32]byte]models.Attestation)
	proposalSate.Sig[round][mypHash][myNodeId] = myAttestation
	proposalSate.SigLock.Unlock()

	// 写入打分
	proposalSate.ScoreLock.Lock()
	proposalSate.Score[round][mypHash] = models.Score{
		Score:     0,
		LastScore: 0,
	}
	proposalSate.ScoreLock.Unlock()

	// 将自己的提案加入处理队列
	select {
	case pChan <- mypHash:
	default:
	}

	// 循环外部变量
	firstSend := false
	nextRound := TimeNextRound(interval, round+1)

	// 处理循环
	for {
		if TimeNextRoundComing(interval, round+1) {
			var winner [32]byte
			var no1 uint32 = 0

			// 选出分数第一
			proposalSate.ScoreLock.RLock()
			for k, v := range proposalSate.Score[round] {
				logger.Debug("choose: %v score: %v", k, v.Score)
				if v.Score >= no1 {
					winner = k
					no1 = v.Score
				}
			}
			proposalSate.ScoreLock.RUnlock()

			logger.Info("round: %v winner: %v", round, winner)

			// 克隆胜利提案的结构体
			proposalSate.DataLock.RLock()
			wp := cloneProposalBody(proposalSate.Data[round][winner])
			proposalSate.DataLock.RUnlock()

			// 将胜利提案写入文件
			if err := winnerProposal(wp, round); err != nil {
				return fmt.Errorf("write winner proposal failed: %v", err)
			}

			return nil
		}

		select {
		case proposalHash := <-pChan:

			proposalSate.DataLock.RLock()
			if _, e := proposalSate.Data[round][proposalHash]; !e {
				proposalSate.DataLock.RUnlock()
				continue
			}
			proposalSate.DataLock.RUnlock()

			proposalSate.SigLock.RLock()
			sigList := cloneAttestationMap(proposalSate.Sig[round][proposalHash])
			proposalSate.SigLock.RUnlock()

			score, err := RateScore(sigList, db.GetDB())
			if err != nil {
				logger.Error("Failed to calculate score: %v", err)
				continue
			}

			// 获取分数
			proposalSate.ScoreLock.Lock()
			stateScore, ok := proposalSate.Score[round][proposalHash]

			if !ok {
				stateScore = models.Score{
					LastScore: 0,
				}
			}
			stateScore.Score = score

			if proposalHash == mypHash {
				if !firstSend || score-stateScore.LastScore >= ScoreBurrs {
					// 写入 Last
					stateScore.LastScore = score
					proposalSate.Score[round][proposalHash] = stateScore
					proposalSate.ScoreLock.Unlock()

					// 广播
					err := sendProposal(mainState, round, proposalHash, db.GetDB(), DiffuseHop)
					if err != nil {
						logger.Error("failed to send my proposal: %v", err)
						continue
					}
					firstSend = true
				} else {
					proposalSate.Score[round][proposalHash] = stateScore
					proposalSate.ScoreLock.Unlock()
				}
			} else if score-stateScore.LastScore >= ScoreBurrs {

				// 写入 Last
				stateScore.LastScore = score
				proposalSate.Score[round][proposalHash] = stateScore
				proposalSate.ScoreLock.Unlock()

				// 创建打分签名
				nodeId, att, errA := buildRateSig(round, proposalHash, score)
				if errA != nil {
					logger.Error("failed to build rate signature: %v", errA)
					continue
				}

				// 写入打分签名
				proposalSate.SigLock.Lock()
				if _, e := proposalSate.Sig[round][proposalHash]; !e {
					proposalSate.Sig[round][proposalHash] = make(map[[32]byte]models.Attestation)
				}
				proposalSate.Sig[round][proposalHash][nodeId] = att
				proposalSate.SigLock.Unlock()

				// 广播提案
				errB := sendProposal(mainState, round, proposalHash, db.GetDB(), DiffuseHop)
				if errB != nil {
					logger.Error("failed to send proposal: %v", errB)
					continue
				}
			} else {
				proposalSate.Score[round][proposalHash] = stateScore
				proposalSate.ScoreLock.Unlock()
			}
		case <-nextRound:
			continue
		}
	}
}
