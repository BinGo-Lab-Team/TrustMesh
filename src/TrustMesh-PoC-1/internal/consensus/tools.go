package consensus

import (
	"TrustMesh-PoC-1/internal/constants"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/table"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

type storedProposal struct {
	ProposerPubKey string `json:"proposer_pub_key"`
	Payload        string `json:"payload"`
	Timestamp      string `json:"timestamp"`
	ProposerSig    string `json:"proposer_sig"`
}

func cloneAttestationMap(src map[[32]byte]models.Attestation) map[[32]byte]models.Attestation {
	dst := make(map[[32]byte]models.Attestation, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneGuaranteeMapMap(src map[[32]byte]map[[32]byte]models.Guarantee) map[[32]byte]map[[32]byte]models.Guarantee {
	dst := make(map[[32]byte]map[[32]byte]models.Guarantee, len(src))

	for outerKey, innerMap := range src {
		if innerMap == nil {
			dst[outerKey] = nil
			continue
		}

		copiedInner := make(map[[32]byte]models.Guarantee, len(innerMap))
		for innerKey, guarantee := range innerMap {
			copiedInner[innerKey] = guarantee
		}

		dst[outerKey] = copiedInner
	}

	return dst
}

func cloneProposalBody(src models.ProposalBody) models.ProposalBody {
	dst := models.ProposalBody{
		ProposerPubKey: src.ProposerPubKey,
		Timestamp:      src.Timestamp,
		ProposerSig:    src.ProposerSig,
	}

	// 对 Payload 做深拷贝
	if src.Payload != nil {
		dst.Payload = make([]byte, len(src.Payload))
		copy(dst.Payload, src.Payload)
	}

	return dst
}

func proposalPath() string {
	path := filepath.Join(constants.ConfigDir, "block")
	return path
}

func winnerProposal(p models.ProposalBody, round int64) error {
	// 转换
	stored := storedProposal{
		ProposerPubKey: hex.EncodeToString(p.ProposerPubKey[:]),
		Payload:        string(p.Payload),
		Timestamp:      time.UnixMilli(int64(p.Timestamp)).UTC().Format(time.RFC3339),
		ProposerSig:    hex.EncodeToString(p.ProposerSig[:]),
	}

	// 序列化
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(proposalPath(), 0o755); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}

	// 文件路径
	filePath := filepath.Join(proposalPath(), fmt.Sprintf("%d.json", round))

	// 写入文件
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}

func randomReputation(r *rand.Rand) uint32 {
	mean := 5000.0
	sigma := 1500.0
	lower, upper := MinReputation, MaxReputation

	for {
		v := int(r.NormFloat64()*sigma + mean)
		if v >= lower && v <= upper {
			return uint32(v)
		}
	}
}

// RandomDBReputation 随机数据库信誉值
func RandomDBReputation(db *gorm.DB) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	var peers []table.Peer
	if err := db.Find(&peers).Error; err != nil {
		return err
	}

	batchSize := 100
	total := len(peers)

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		for j := i; j < end; j++ {
			newRep := randomReputation(r)

			if err := db.Model(&peers[j]).Update("reputation", newRep).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
