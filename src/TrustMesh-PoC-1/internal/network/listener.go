package network

import (
	"TrustMesh-PoC-1/internal/db"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/p2p"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// StartNodeServer 启动节点监听服务
func StartNodeServer(mainState *models.MainStore) error {
	// NODE_PORT 是否合法
	nodePort, isExist := os.LookupEnv("NODE_PORT")
	if isExist == false {
		return fmt.Errorf("NODE_PORT is missing")
	} else if err := parsePort(nodePort); err != nil {
		return fmt.Errorf("invalid NODE_PORT: %w", err)
	}

	addr := ":" + nodePort
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Info("node is listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Warning("Connection Error: %v", err)
			continue
		}
		ready := make(chan [32]byte)
		go p2p.HandleConnection(conn, mainState, ready, false)
	}
}

// StartBootstrapServer 启动引导节点服务
func StartBootstrapServer(mainState *models.MainStore) error {
	// NODE_PORT 是否合法
	nodePort, isExist := os.LookupEnv("NODE_PORT")
	if isExist == false {
		return fmt.Errorf("NODE_PORT is missing")
	} else if err := parsePort(nodePort); err != nil {
		return fmt.Errorf("invalid NODE_PORT: %w", err)
	}

	addr := ":" + nodePort
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Info("Bootstrap is listening on %s", addr)

	// 初始化节点列表
	nl := p2p.NodeList{
		Node: make(map[[32]byte]string),
	}

	// 获取等待时间
	var waitTime time.Duration
	if val, exists := os.LookupEnv("WAIT_TIME"); exists {
		i, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("WAIT_TIME is not a number")
		} else if i < 0 {
			return fmt.Errorf("WAIT_TIME must >= 0")
		}
		waitTime = time.Duration(i) * time.Second
	} else {
		return fmt.Errorf("WAIT_TIME is missing")
	}

	timer := time.NewTimer(waitTime)
	stop := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-stop:
					return
				default:
					logger.Warning("Accept error: %v", err)
					continue
				}
			}
			go p2p.BootstrapHandleConnection(conn, &nl, mainState)
		}
	}()

	// 主控方
	select {
	case <-timer.C:
		close(stop)
		if err := listener.Close(); err != nil {
			return err
		}
	}

	var n int
	if val, exists := os.LookupEnv("DENSITY"); exists {
		i, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("DENSITY is not a number")
		} else if i <= 0 {
			return fmt.Errorf("DENSITY must > 0")
		}
		n = i
	} else {
		return fmt.Errorf("DENSITY is missing")
	}

	nl.Mu.Lock()
	nodeList := nl.Node

	replyAddr := make(map[[32]byte][]byte, len(nodeList))
	queue := make([][32]byte, 0, len(nodeList)*n)

	o := 0
	for k := range nodeList {
		for i := 0; i < n; i++ {
			queue = append(queue, k)
			o++
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })

	// 分配地址
	idx := 0
	for k := range nodeList {
		var v [][]byte
		var data []byte
		retry := 0
		for j := 0; j < n && idx < len(queue); {
			msg := make([]byte, 0, 32)

			msg = append(msg, queue[idx][:]...)
			msg = append(msg, '+')
			msg = append(msg, []byte(nodeList[queue[idx]])...)
			msg = append(msg, ';')
			flag := true

			// 检查冲突
			for _, vV := range v {
				if bytes.Equal(msg, vV) || queue[idx] == k {
					flag = false
					break
				}
			}

			// 如果不冲突执行下一个，冲突就将当前项移到末尾并重试
			if flag {
				v = append(v, msg)
				j++
				idx++
				retry = 0
			} else {
				moveToEnd(queue, idx)
				retry++
			}

			// 如果重试次数过多就强制退出
			if retry >= 100 {
				break
			}
		}
		for _, kV := range v {
			data = append(data, kV...)
		}
		replyAddr[k] = append(replyAddr[k], data...)
	}

	// 互认
	for k, data := range replyAddr {
		for i := 0; i < len(data); {
			var peerID [32]byte
			copy(peerID[:], data[i:i+32])

			// 检查 peerID 是否已包含 k
			if !containsPeer(replyAddr[peerID], k) {
				addr := nodeList[k]
				peerMsg := make([]byte, 0, 32)

				peerMsg = append(peerMsg, k[:]...)
				peerMsg = append(peerMsg, '+')
				peerMsg = append(peerMsg, addr...)
				peerMsg = append(peerMsg, ';')
				replyAddr[peerID] = append(replyAddr[peerID], peerMsg...)
			}

			i += 32
			// 找到 ';'
			for i < len(data) && data[i] != ';' {
				i++
			}
			i++ // 跳过 ';'
		}
	}
	nl.Mu.Unlock()

	var wg sync.WaitGroup
	for k, data := range replyAddr {
		var message []byte
		// 创建 [4]byte 格式的 header
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], p2p.MsgBootstrapReply)
		// 创建 [4]byte 格式的 length
		if len(data) > math.MaxInt32 {
			logger.Warning("Message too large (%d bytes)", len(data))
			continue
		}
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(data)))
		message = append(message, header[:]...)
		message = append(message, length[:]...)
		message = append(message, data...)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok := p2p.SendNodeIdMessage(k, message, db.GetDB(), mainState); !ok {
				logger.Warning("Failed to send to: %v message: %v", k, message)
			}
		}()
	}

	wg.Wait()

	logger.Info("All work is done")

	return nil
}
