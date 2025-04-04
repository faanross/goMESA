package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"goMESA/internal/server"
	"goMESA/internal/server/api"
	"log"
	"net/http"
	"os"
)

func main() {
	// Parse command line flags - only need the database path now
	var dbPath string
	flag.StringVar(&dbPath, "path", "mesa.db", "Path to SQLite database file")
	flag.Parse()

	// Validate dbPath (optional but good practice)
	if dbPath == "" {
		log.Fatalf("Database path cannot be empty. Please use the -path flag.")
	}

	// Check if running as root/administrator (required for NTP server raw socket/port 123)
	if os.Geteuid() != 0 {
		fmt.Println("[!] This program requires root/administrator privileges to run")
		fmt.Println("[!] Please restart with sudo or as administrator")
		os.Exit(1)
	}

	// --- MySQL logic removed ---

	// Create SQLite database instance directly
	// The NewDatabase function now only takes the path
	db, err := server.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database at '%s': %v", dbPath, err)
	}
	// Ensure the database connection is closed when main exits
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Create NTP server, passing the database instance
	ntpServer := server.NewNTPServer(db) // Assumes NewNTPServer exists and takes *Database
	err = ntpServer.Start()
	if err != nil {
		// Attempt to close DB before fatal exit if NTP server fails
		_ = db.Close()
		log.Fatalf("Failed to start NTP server: %v", err)
	}
	// Note: Stopping the NTP server gracefully on interrupt (Ctrl+C) is not handled here,
	// but could be added using signal handling.

	log.Println("TUI exited.") // Log when TUI finishes
	// NTP server stopping is handled by defer db.Close() implicitly if needed,
	// or explicitly add ntpServer.Stop() if required before exit.
}
