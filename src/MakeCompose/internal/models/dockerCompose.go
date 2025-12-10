package models

const (
	Image   = "bingolab/trustmesh-node:poc-1.0.0"
	Network = "trustmesh-net"
)

// DockerCompose 主结构体
type DockerCompose struct {
	Services map[string]any `yaml:"services"`
	Networks map[string]any `yaml:"networks"`
}

// Init 初始化 DockerCompose 结构体
func (cfg *DockerCompose) Init() {
	*cfg = DockerCompose{
		Services: make(map[string]any),
		Networks: make(map[string]any),
	}
}
