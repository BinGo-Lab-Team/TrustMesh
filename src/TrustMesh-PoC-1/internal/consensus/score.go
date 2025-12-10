package consensus

import (
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/table"
	"fmt"

	"gorm.io/gorm"
)

// RateScore 打分
func RateScore(sigList map[[32]byte]models.Attestation, db *gorm.DB) (uint32, error) {
	var score uint32 = 0
	var total uint64

	// 获取总值
	err := db.Model(&table.Peer{}).
		Select("COALESCE(SUM(reputation), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("error reputation statistics failed: %v", err)
	}

	if total == 0 {
		return 0, fmt.Errorf("all the reputation data are zero")
	}

	for k, v := range sigList {
		if v.Score > MaxScore {
			// TODO: 最大限制
			// 由于 PoC 的分数可能会经常变动，因此不加最大限制，有溢出风险注意
			logger.Warning("max score exceeded")
			//continue
		}

		var proportion float64
		var reputation uint32
		err := db.Model(&table.Peer{}).
			Select("reputation").
			Where("node_id = ?", k[:]).
			Take(&reputation).Error
		if err != nil {
			continue
		}

		proportion = float64(reputation) / float64(total)

		score += uint32(float64(v.Score) * proportion)
	}

	return score, nil
}
