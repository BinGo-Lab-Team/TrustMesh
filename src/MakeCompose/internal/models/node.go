package models

// Node 节点结构体
type Node struct {
	Image       string            `yaml:"image"`
	Environment map[string]string `yaml:"environment"`
	Volumes     []string          `yaml:"volumes"`
	Networks    []string          `yaml:"networks"`
}
