package main

import (
	"log"
	"os"

	"github.com/Varun03-max/JIO/cmd"
)

func main() {
	// Read port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Force config load from environment variables only
	// by setting ConfigPath to empty string
	serverConfig := cmd.JioTVServerConfig{
		Host:       "0.0.0.0",
		Port:       port,
		ConfigPath: "",     // Always load from ENV, never file
		TLS:        false,  // Change to true if using HTTPS
	}

	// Start the server
	if err := cmd.JioTVServer(serverConfig); err != nil {
		log.Fatalf("Failed to start JioTV server: %v", err)
	}
}
