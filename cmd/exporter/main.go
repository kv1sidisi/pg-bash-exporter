package main

import (
	"flag"
	"fmt"
	"pg-bash-exporter/internal/config"
)

func main() {
	flag.Parse()

	configPath := config.GetPath()

	var cfg config.Config

	if err := config.Load(configPath, &cfg); err != nil {
		fmt.Printf("failed to load configuration: %v", err)
	}

	fmt.Printf("scdsc")
}
