package main

import (
	"flag"
	"fmt"
	"log"
	"pg-bash-exporter/internal/config"
)

func main() {
	flag.Parse()

	configPath := config.GetPath()

	var cfg config.Config

	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatal("failed to load configuration: %v", err)
	}

	fmt.Printf("scdsc")
}
