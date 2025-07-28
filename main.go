package main

import (
	"log"
	"os"

	"github.com/Varun03-max/JIO/cmd"
)

func main() {
	// Read port and config from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	serverConfig := cmd.JioTVServerConfig{
		Host:       "0.0.0.0",
		Port:       port,
		ConfigPath: configPath,
		TLS:        false, // or true if you're using HTTPS
	}

	if err := cmd.JioTVServer(serverConfig); err != nil {
		log.Fatalf("Failed to start JioTV server: %v", err)
	}
}
