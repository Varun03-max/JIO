package main

import (
	"log"
	"os"

	"github.com/Varun03-max/JIO/cmd"
)

func main() {
	// Read port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Optional config path — use empty to load from ENV
	configPath := os.Getenv("CONFIG_PATH")

	// If CONFIG_PATH is empty, we use environment variables (no file needed)
	serverConfig := cmd.JioTVServerConfig{
		Host:       "0.0.0.0",
		Port:       port,
		ConfigPath: configPath, // "" triggers env loading in your config.go
		TLS:        false,
	}

	if err := cmd.JioTVServer(serverConfig); err != nil {
		log.Fatalf("Failed to start JioTV server: %v", err)
	}
}
