package interaction

import (
	"MakeCompose/internal/models"
	"fmt"
	"path/filepath"
	"strconv"
)

// WriteDockerCompose 交互式询问并写入信息
func WriteDockerCompose(cfg *models.DockerCompose) error {
	var nodeCount int
	var needBootstrap bool

	var port int
	var debugMode bool
	var testMode bool
	var interval int
	var folder string

	// 存储文件夹
	folder = Text("Enter volumes folder")

	// 端口
	for {
		port = Number("Enter port number")
		if port >= 0 && port <= 65535 {
			break
		} else {
			fmt.Println("Please enter a valid port number (0~65535)")
		}
	}

	// 轮次间隔
	for {
		interval = Number("Enter round interval (seconds)")
		if interval > 0 {
			break
		} else {
			fmt.Println("Please enter a valid interval (>0)")
		}
	}

	// DEBUG 级别
	debugMode = YesOrNo("Enable debug mode?")

	// TEST 级别
	testMode = YesOrNo("Enable test mode?")

	// 节点数量
	nodeCount = Number("How many nodes do you want to create?")

	// 引导节点
	needBootstrap = YesOrNo("Do you need bootstrap node?")
	if needBootstrap {
		var waitTime int

		// 等待时间
		waitTime = Number("Enter bootstrap wait time (seconds)")
		if waitTime < 0 {
			waitTime = 0
		}

		// 邻居密度
		var density int
		for {
			density = Number("Enter density")
			if density > 0 {
				break
			} else {
				fmt.Println("Please enter a valid density (>0)")
			}
		}

		// 环境变量
		environment := make(map[string]string)
		environment["INSTANCE_ID"] = "0"
		environment["NODE_PORT"] = strconv.Itoa(port)
		environment["DEBUG_MODE"] = BoolToString(debugMode)
		environment["TEST_MODE"] = BoolToString(testMode)
		environment["HOST"] = "bootstrap:" + strconv.Itoa(port)
		environment["WAIT_TIME"] = strconv.Itoa(waitTime)
		environment["DENSITY"] = strconv.Itoa(density)

		// 存储
		volumes := make([]string, 0, 1)
		volumes = append(volumes, NormalizeForCompose(filepath.Join(folder, "bootstrap"))+":/data")

		// 网络
		networks := make([]string, 0, 1)
		networks = append(networks, models.Network)

		node := models.Node{
			Image:       models.Image,
			Environment: environment,
			Volumes:     volumes,
			Networks:    networks,
		}

		cfg.Services["bootstrap"] = node
	}

	// 普通节点
	for i := 1; i <= nodeCount; i++ {
		nodeName := fmt.Sprintf("node-%d", i)

		// 环境变量
		environment := make(map[string]string)
		environment["INSTANCE_ID"] = strconv.Itoa(i)
		environment["NODE_PORT"] = strconv.Itoa(port)
		environment["DEBUG_MODE"] = BoolToString(debugMode)
		environment["TEST_MODE"] = BoolToString(testMode)
		environment["HOST"] = nodeName + ":" + strconv.Itoa(port)
		environment["INTERVAL"] = strconv.Itoa(interval)

		// 引导节点地址
		if needBootstrap {
			environment["BOOTSTRAP"] = "bootstrap:" + strconv.Itoa(port)
		}

		// 存储
		volumes := make([]string, 0, 1)
		volumes = append(volumes, NormalizeForCompose(filepath.Join(folder, nodeName))+":/data")

		// 网络
		networks := make([]string, 0, 1)
		networks = append(networks, models.Network)

		node := models.Node{
			Image:       models.Image,
			Environment: environment,
			Volumes:     volumes,
			Networks:    networks,
		}

		cfg.Services[nodeName] = node
	}

	cfg.Networks[models.Network] = models.Net{}

	return nil
}
