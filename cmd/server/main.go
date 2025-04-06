package main

import (
	"flag"
	"fmt"
	"goMESA/internal/server"
	"goMESA/internal/server/api"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Parse command line flags
	var dbPath string
	var apiPort int
	flag.StringVar(&dbPath, "path", "./data/gomesa.db", "Path to SQLite database file")
	flag.IntVar(&apiPort, "port", 8080, "Port for the web API server")
	flag.Parse()

	// Validate dbPath
	if dbPath == "" {
		log.Fatalf("Database path cannot be empty. Please use the -path flag.")
	}

	// Ensure the directory for the database exists
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory '%s': %v", dataDir, err)
	}

	// Check if running as root/administrator (required for NTP server raw socket/port 123)
	if os.Geteuid() != 0 {
		fmt.Println("[!] This program requires root/administrator privileges to run")
		fmt.Println("[!] Please restart with sudo or as administrator")
		os.Exit(1)
	}

	// Create SQLite database instance
	db, err := server.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database at '%s': %v", dbPath, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Create NTP server
	ntpServer := server.NewNTPServer(db)
	err = ntpServer.Start()
	if err != nil {
		_ = db.Close()
		log.Fatalf("Failed to start NTP server: %v", err)
	}

	// Create and start API server
	apiServer := api.NewAPIServer(db, ntpServer, apiPort)
	log.Printf("Starting API server on port %d...", apiPort)
	if err := apiServer.Start(); err != nil {
		log.Fatalf("Error running API server: %v", err)
	}
}
