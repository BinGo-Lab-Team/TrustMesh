package network

import (
	"TrustMesh-PoC-1/internal/consensus"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/p2p"
	"TrustMesh-PoC-1/internal/tools"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// StartRequestList 向引导节点请求节点列表
func StartRequestList(mainState *models.MainStore) error {
	ct := mainState.ConnectionTable

	var localAddr []byte
	if host, isExist := os.LookupEnv("HOST"); !isExist {
		return fmt.Errorf("NODE_PORT is missing")
	} else {
		localAddr = []byte(host)
	}

	bootstrap, isExist := os.LookupEnv("BOOTSTRAP")
	if !isExist {
		return fmt.Errorf("BOOTSTRAP is missing")
	}

	rawConn, err := net.Dial("tcp", bootstrap)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], p2p.MsgBootstrapReport)
	var length [4]byte
	// !!! 截断风险 !!!
	binary.BigEndian.PutUint32(length[:], uint32(len(localAddr)))

	var message []byte
	message = append(message, header[:]...)
	message = append(message, length[:]...)
	message = append(message, localAddr...)

	ready := make(chan [32]byte, 1)
	done := make(chan struct{})

	go p2p.HandleConnection(rawConn, mainState, ready, true)

	select {
	case realId := <-ready:
		close(done)

		ct.Lock.RLock()
		newConn, ok := ct.Connection[realId]
		ct.Lock.RUnlock()

		if !ok {
			return fmt.Errorf("connection not found")
		}

		select {
		case newConn.WriteQueue <- message:
			return nil
		default:
			return fmt.Errorf("failed to send")
		}
	case <-tools.WaitTimeout(done, 5*time.Second):
		return fmt.Errorf("send timeout")
	}
}

// StartNodeClient 启动节点主动服务
func StartNodeClient(mainState *models.MainStore) error {
	var round int64 = 0
	var interval time.Duration

	// 提取环境变量里的轮次间隔
	if val, exists := os.LookupEnv("INTERVAL"); exists {
		num, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("WAIT_TIME is not a number")
		} else if num <= 0 {
			return fmt.Errorf("WAIT_TIME must > 0")
		}
		interval = time.Duration(num) * time.Second
	} else {
		return fmt.Errorf("INTERVAL is missing")
	}

	// 轮次循环
	for {
		select {
		case round = <-consensus.TimeRoundEngine(interval, round):
			go func(r int64) {
				logger.Info("New round: %v", r)
				if err := consensus.ExecuteRound(mainState, r, interval); err != nil {
					logger.Error("ExecuteRound error: %v", err)
				}
			}(round)
			round++
		}
	}
}
