package main

import (
	"MakeCompose/internal/interaction"
	"MakeCompose/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	var cfg models.DockerCompose
	cfg.Init()

	err := interaction.WriteDockerCompose(&cfg)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("docker-compose.yml")
	if err != nil {
		panic(err)
	}

	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)

	err = enc.Encode(cfg)
	if err != nil {
		panic(err)
	}

	err = enc.Close()
	if err != nil {
		panic(err)
	}

	path, err := filepath.Abs(file.Name())
	if err != nil {
		panic(err)
	}

	fmt.Printf("The docker-compose is created at: %s", path)
}
