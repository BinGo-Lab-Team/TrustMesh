package main

import (
	"TrustMesh-PoC-1/internal/db"
	"TrustMesh-PoC-1/internal/keys"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/models"
	"TrustMesh-PoC-1/internal/network"
	"TrustMesh-PoC-1/internal/node"
	"TrustMesh-PoC-1/internal/p2p"
	"TrustMesh-PoC-1/internal/tools"
	"os"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

func main() {
	// *唯一* 启动/关闭 提示
	logger.Info("TrustMesh PoC Started")
	defer logger.Info("TrustMesh PoC Stopped")

	_, pk, err := keys.LoadOrCreateKey()
	if err != nil {
		logger.Error("Error loading keys: ", err)
	} else {
		logger.Info("My NodeId: %v", blake3.Sum256(pk[:]))
	}

	// 退出信号
	quit := make(chan struct{})
	var quitOnce sync.Once

	defer quitOnce.Do(func() { close(quit) })

	flag, err := tools.EnsureFilePath(db.Path())
	if err != nil {
		logger.Fatal("Database path error: ", err)
		return
	}

	// 初始化数据库
	if _, err := db.InitDB(); err != nil {
		logger.Fatal("Error connecting to database: %v", err)
		return
	}

	// 初始化 P2P
	p2p.Init(node.Node{})

	// 创建主存储变量
	var mainState models.MainStore
	mainState.Init()

	// 启动服务
	if id, exist := os.LookupEnv("INSTANCE_ID"); !exist {
		logger.Fatal("INSTANCE_ID not set")
		return
	} else if id == "0" {
		logger.Info("Start Bootstrap Server")

		// 启动引导节点服务
		go func() {
			if err := network.StartBootstrapServer(&mainState); err != nil {
				logger.Fatal("Error starting bootstrap server")
			}
			time.Sleep(5 * time.Second)
			quitOnce.Do(func() { close(quit) })
		}()
	} else {
		logger.Info("Start node Server")

		start := make(chan struct{})

		// 请求节点列表
		if !flag {
			go func() {
				time.Sleep(1 * time.Second)
				logger.Info("Start Request List")
				if err := network.StartRequestList(&mainState); err != nil {
					logger.Error(err.Error())
					return
				}
			}()
		} else {
			close(start)
		}

		// 启动服务端服务
		go func() {
			if err := network.StartNodeServer(&mainState); err != nil {
				logger.Fatal("Error starting node server")
				return
			}
		}()

		// 开始前阻塞
		<-start

		// 启动客户端服务
		go func() {
			if err := network.StartNodeClient(&mainState); err != nil {
				logger.Fatal("Error starting node client: %v", err)
			}
		}()
	}

	// 监听退出信号
	select {
	case <-quit:
		return
	}
}
