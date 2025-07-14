package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"pg-bash-exporter/internal/config"
)

var ValidationFlag bool

func init() {
	flag.BoolVar(&ValidationFlag, "validate-config", false, "Validate the configuration file.")
}

func main() {
	flag.Parse()

	configPath := config.GetPath()

	if ValidationFlag {
		var cfg config.Config

		fmt.Println("Validating configuration file:", configPath)

		if err := config.Load(configPath, &cfg); err != nil {
			log.Fatalf("configuration is invalid: %v", err)
		}

		fmt.Println("Configuration is valid.")

		os.Exit(0)
	}

	var cfg config.Config

	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	fmt.Printf("scdsc")
}
