package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Varun03-max/JIO/cmd"
)

func main() {
	// Load the configuration
	config, err := cmd.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize the logger
	cmd.InitializeLogger(config)

	// Use the logger
	cmd.Logger.Info("Starting JioTV Go CLI")

	// Start the CLI app
	err = cmd.RunCLI()
	if err != nil {
		cmd.Logger.Fatalf("CLI Error: %v", err)
	}

	// Setup for Render or local server if needed
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	host := "0.0.0.0"
	addr := fmt.Sprintf("%s:%s", host, port)

	cmd.Logger.Infof("Web server running on %s", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		cmd.Logger.Fatalf("Server failed: %v", err)
	}
}
