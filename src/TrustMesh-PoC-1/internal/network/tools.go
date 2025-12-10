package network

import (
	"fmt"
	"strconv"
)

// parsePort 检查端口字符串是否合法
func parsePort(strPort string) error {
	// 字符串是否为空
	if strPort == "" {
		return fmt.Errorf("port is empty")
	}

	// 字符串是否为整数
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return fmt.Errorf("invalid port %s: %w", strPort, err)
	}

	// 端口范围是否合法
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range", port)
	}

	return nil
}

// moveToEnd 将选中值移动到队尾
func moveToEnd(queue [][32]byte, idx int) [][32]byte {
	if idx < 0 || idx >= len(queue) {
		return queue // 越界保护
	}

	v := queue[idx]
	copy(queue[idx:], queue[idx+1:]) // 所有后面元素左移
	queue[len(queue)-1] = v

	return queue
}

// containsPeer 检查指定节点在切片里是否存在
func containsPeer(data []byte, peer [32]byte) bool {
	for i := 0; i < len(data); {
		var id [32]byte
		copy(id[:], data[i:i+32])
		if id == peer {
			return true
		}

		i += 32
		for i < len(data) && data[i] != ';' {
			i++
		}
		i++
	}
	return false
}
