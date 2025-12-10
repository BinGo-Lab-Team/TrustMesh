package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BlockData struct {
	ProposerPubKey string `json:"proposer_pub_key"`
}

func FindBlockDirs(root string) (map[string]string, error) {
	result := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否是名为 "block" 的文件夹
		if info.IsDir() && info.Name() == "block" {
			parent := filepath.Base(filepath.Dir(path))
			result[parent] = path
		}

		return nil
	})

	return result, err
}

func CountProposalsForRound(blockDirs map[string]string, round int) (map[string]int, error) {
	result := make(map[string]int)

	filename := fmt.Sprintf("%d.json", round)

	for nodeName, blockPath := range blockDirs {
		filePath := filepath.Join(blockPath, filename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			// block 不存在表示该节点该轮没有 block，直接跳过即可
			continue
		}

		var block BlockData
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("节点 %s JSON 解析失败: %v", nodeName, err)
		}

		// 防御性处理
		if len(block.ProposerPubKey) < 5 {
			continue
		}

		// 使用前五个字符标记提案
		proposalID := block.ProposerPubKey[:5]

		result[proposalID]++
	}

	return result, nil
}
