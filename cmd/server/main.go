package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"goMESA/internal/server"

	"golang.org/x/term"
)

func main() {
	// Parse command line flags
	var (
		dbType   string
		dbPath   string
		username string
		password string
	)

	flag.StringVar(&dbType, "db", "sqlite", "Database type (sqlite or mysql)")
	flag.StringVar(&dbPath, "path", "mesa.db", "Path to SQLite database file")
	flag.StringVar(&username, "user", "", "MySQL username")
	flag.Parse()

	// Check if root/admin
	if os.Geteuid() != 0 {
		fmt.Println("[!] This program requires root/administrator privileges to run")
		fmt.Println("[!] Please restart with sudo or as administrator")
		os.Exit(1)
	}

	// If MySQL, prompt for credentials if not provided
	if dbType == "mysql" {
		if username == "" {
			fmt.Print("Enter MySQL username: ")
			fmt.Scanln(&username)
		}

		fmt.Print("Enter MySQL password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
		}
		password = string(passwordBytes)
		fmt.Println() // Add a newline after password input
	}

	// Create database instance
	var db *server.Database
	var err error

	switch dbType {
	case "sqlite":
		db, err = server.NewDatabase(server.SQLite, dbPath, "", "")
	case "mysql":
		db, err = server.NewDatabase(server.MySQL, "", username, password)
	default:
		log.Fatalf("Unsupported database type: %s", dbType)
	}

	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create NTP server
	ntpServer := server.NewNTPServer(db)
	err = ntpServer.Start()
	if err != nil {
		log.Fatalf("Failed to start NTP server: %v", err)
	}

	// Create and start TUI
	tui := server.NewTUI(db, ntpServer)
	if err := tui.Start(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
}
