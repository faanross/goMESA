package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"goMESA/internal/common"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// DBType represents the database type
type DBType int

const (
	// MySQL database
	MySQL DBType = iota
	// SQLite database - more portable than MySQL
	SQLite
)

// Database handles all database operations
type Database struct {
	db       *sql.DB
	dbType   DBType
	dbPath   string
	username string
	password string
}

// NewDatabase creates a new database instance
func NewDatabase(dbType DBType, dbPath, username, password string) (*Database, error) {
	database := &Database{
		dbType:   dbType,
		dbPath:   dbPath,
		username: username,
		password: password,
	}

	err := database.Connect()
	if err != nil {
		return nil, err
	}

	return database, nil
}

// Connect establishes a connection to the database
func (d *Database) Connect() error {
	var err error
	var dataSourceName string

	switch d.dbType {
	case MySQL:
		// First connect without selecting a database to create it if it doesn't exist
		dataSourceName = fmt.Sprintf("%s:%s@tcp(localhost:3306)/", d.username, d.password)
		d.db, err = sql.Open("mysql", dataSourceName)
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %v", err)
		}

		// Create the database if it doesn't exist
		_, err = d.db.Exec("CREATE DATABASE IF NOT EXISTS mesaC2")
		if err != nil {
			return fmt.Errorf("failed to create database: %v", err)
		}

		// Close the initial connection
		d.db.Close()

		// Connect to the specific database
		dataSourceName = fmt.Sprintf("%s:%s@tcp(localhost:3306)/mesaC2", d.username, d.password)
		d.db, err = sql.Open("mysql", dataSourceName)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}
	case SQLite:
		// Connect to SQLite
		d.db, err = sql.Open("sqlite3", d.dbPath)
		if err != nil {
			return fmt.Errorf("failed to connect to SQLite: %v", err)
		}
	}

	// Test the connection
	err = d.db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create tables if they don't exist
	err = d.createTables()
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	log.Println("Database connection established")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// createTables creates the necessary tables if they don't exist
func (d *Database) createTables() error {
	var agentsTable, commandsTable string

	switch d.dbType {
	case MySQL:
		agentsTable = `
		CREATE TABLE IF NOT EXISTS agents (
			agent_id VARCHAR(45) NOT NULL PRIMARY KEY,
			os VARCHAR(255),
			service VARCHAR(255),
			status VARCHAR(20) NOT NULL DEFAULT 'ALIVE',
			first_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			network_adapter VARCHAR(255)
		)
		`
		commandsTable = `
		CREATE TABLE IF NOT EXISTS commands (
			id INT AUTO_INCREMENT PRIMARY KEY,
			agent_id VARCHAR(45) NOT NULL,
			content TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(20) NOT NULL DEFAULT 'SENT',
			output TEXT,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
		)
		`
	case SQLite:
		agentsTable = `
		CREATE TABLE IF NOT EXISTS agents (
			agent_id TEXT NOT NULL PRIMARY KEY,
			os TEXT,
			service TEXT,
			status TEXT NOT NULL DEFAULT 'ALIVE',
			first_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			network_adapter TEXT
		)
		`
		commandsTable = `
		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status TEXT NOT NULL DEFAULT 'SENT',
			output TEXT,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
		)
		`
	}

	// Create agents table
	_, err := d.db.Exec(agentsTable)
	if err != nil {
		return fmt.Errorf("failed to create agents table: %v", err)
	}

	// Create commands table
	_, err = d.db.Exec(commandsTable)
	if err != nil {
		return fmt.Errorf("failed to create commands table: %v", err)
	}

	return nil
}

// AddAgent adds a new agent to the database
func (d *Database) AddAgent(agentID, networkAdapter string) error {
	// Check if agent already exists
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM agents WHERE agent_id = ?", agentID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if agent exists: %v", err)
	}

	if count > 0 {
		// Update the existing agent's last_seen timestamp and status
		_, err = d.db.Exec(
			"UPDATE agents SET last_seen = ?, status = ?, network_adapter = ? WHERE agent_id = ?",
			time.Now(), common.AgentStatusAlive, networkAdapter, agentID,
		)
		return err
	}

	// Insert new agent
	_, err = d.db.Exec(
		"INSERT INTO agents (agent_id, status, network_adapter) VALUES (?, ?, ?)",
		agentID, common.AgentStatusAlive, networkAdapter,
	)
	if err != nil {
		return fmt.Errorf("failed to add agent: %v", err)
	}

	log.Printf("Agent %s added to database", agentID)
	return nil
}

// UpdateAgentStatus updates an agent's status
func (d *Database) UpdateAgentStatus(agentID, status string) error {
	_, err := d.db.Exec(
		"UPDATE agents SET status = ?, last_seen = ? WHERE agent_id = ?",
		status, time.Now(), agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %v", err)
	}
	return nil
}

// SetMissingAgents marks agents as MIA if they haven't been seen for more than 3 minutes
func (d *Database) SetMissingAgents() error {
	var timeThreshold string

	switch d.dbType {
	case MySQL:
		timeThreshold = "DATE_SUB(NOW(), INTERVAL 3 MINUTE)"
	case SQLite:
		timeThreshold = "datetime('now', '-3 minutes')"
	}

	_, err := d.db.Exec(
		fmt.Sprintf("UPDATE agents SET status = ? WHERE last_seen < %s AND status != ?",
			timeThreshold),
		common.AgentStatusMissing, common.AgentStatusKilled,
	)
	if err != nil {
		return fmt.Errorf("failed to set missing agents: %v", err)
	}
	return nil
}

// GetAllAgents returns all agents from the database
func (d *Database) GetAllAgents() ([]common.Agent, error) {
	rows, err := d.db.Query("SELECT agent_id, os, service, status, first_seen, last_seen, network_adapter FROM agents ORDER BY COALESCE(service, '') ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %v", err)
	}
	defer rows.Close()

	var agents []common.Agent
	for rows.Next() {
		var agent common.Agent
		err := rows.Scan(
			&agent.ID,
			&agent.OS,
			&agent.Service,
			&agent.Status,
			&agent.FirstSeen,
			&agent.LastSeen,
			&agent.NetworkAdapter,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %v", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agent rows: %v", err)
	}

	return agents, nil
}

// GetAgentsByGroup returns agents filtered by a group type (OS or service) and value
func (d *Database) GetAgentsByGroup(groupType, value string) ([]common.Agent, error) {
	// Ensure groupType is valid to prevent SQL injection
	if groupType != "os" && groupType != "service" && groupType != "agent_id" {
		return nil, fmt.Errorf("invalid group type: %s", groupType)
	}

	query := fmt.Sprintf("SELECT agent_id, os, service, status, first_seen, last_seen, network_adapter FROM agents WHERE %s = ? ORDER BY COALESCE(service, '') ASC", groupType)
	rows, err := d.db.Query(query, value)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents by group: %v", err)
	}
	defer rows.Close()

	var agents []common.Agent
	for rows.Next() {
		var agent common.Agent
		err := rows.Scan(
			&agent.ID,
			&agent.OS,
			&agent.Service,
			&agent.Status,
			&agent.FirstSeen,
			&agent.LastSeen,
			&agent.NetworkAdapter,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %v", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agent rows: %v", err)
	}

	return agents, nil
}

// AddGrouping adds or updates an OS or service identifier to an agent or range of agents
func (d *Database) AddGrouping(ipRange, groupType, name string) error {
	// Check if this is an IP range
	if strings.Contains(ipRange, "-") {
		// Parse the range
		parts := strings.Split(ipRange, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid IP range format: %s", ipRange)
		}

		// Find the last octet range
		baseIP := parts[0][:strings.LastIndex(parts[0], ".")+1]
		startStr := parts[0][strings.LastIndex(parts[0], ".")+1:]
		endStr := parts[1]

		start, err := fmt.Sscanf(startStr, "%d", new(int))
		if err != nil {
			return fmt.Errorf("invalid start range: %v", err)
		}

		end, err := fmt.Sscanf(endStr, "%d", new(int))
		if err != nil {
			return fmt.Errorf("invalid end range: %v", err)
		}

		// Update each IP in the range
		for i := start; i <= end; i++ {
			ip := fmt.Sprintf("%s%d", baseIP, i)
			_, err := d.db.Exec(
				fmt.Sprintf("UPDATE agents SET %s = ? WHERE agent_id = ?", groupType),
				name, ip,
			)
			if err != nil {
				return fmt.Errorf("failed to update agent %s: %v", ip, err)
			}
		}
		return nil
	}

	// Single IP update
	_, err := d.db.Exec(
		fmt.Sprintf("UPDATE agents SET %s = ? WHERE agent_id = ?", groupType),
		name, ipRange,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent %s: %v", ipRange, err)
	}
	return nil
}

// RemoveAllAgents removes all agents from the database
func (d *Database) RemoveAllAgents() error {
	_, err := d.db.Exec("DELETE FROM agents")
	if err != nil {
		return fmt.Errorf("failed to remove all agents: %v", err)
	}
	return nil
}

// AddCommand adds a new command to the database
func (d *Database) AddCommand(agentID, content string) (int64, error) {
	result, err := d.db.Exec(
		"INSERT INTO commands (agent_id, content, status) VALUES (?, ?, ?)",
		agentID, content, common.CommandStatusSent,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to add command: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get command ID: %v", err)
	}

	return id, nil
}

// UpdateCommandOutput updates a command's output and status
func (d *Database) UpdateCommandOutput(commandID string, output string) error {
	_, err := d.db.Exec(
		"UPDATE commands SET output = ?, status = ? WHERE id = ?",
		output, common.CommandStatusExecuted, commandID,
	)
	if err != nil {
		return fmt.Errorf("failed to update command output: %v", err)
	}
	return nil
}

// GetCommandHistory returns the command history for an agent
func (d *Database) GetCommandHistory(agentID string) ([]common.Command, error) {
	rows, err := d.db.Query(
		"SELECT id, agent_id, content, timestamp, status, output FROM commands WHERE agent_id = ? ORDER BY timestamp DESC",
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query command history: %v", err)
	}
	defer rows.Close()

	var commands []common.Command
	for rows.Next() {
		var cmd common.Command
		err := rows.Scan(
			&cmd.ID,
			&cmd.AgentID,
			&cmd.Content,
			&cmd.Timestamp,
			&cmd.Status,
			&cmd.Output,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command row: %v", err)
		}
		commands = append(commands, cmd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating command rows: %v", err)
	}

	return commands, nil
}

// CleanDatabase removes all tables and recreates them
func (d *Database) CleanDatabase() error {
	// Drop tables
	_, err := d.db.Exec("DROP TABLE IF EXISTS commands")
	if err != nil {
		return fmt.Errorf("failed to drop commands table: %v", err)
	}

	_, err = d.db.Exec("DROP TABLE IF EXISTS agents")
	if err != nil {
		return fmt.Errorf("failed to drop agents table: %v", err)
	}

	// Recreate tables
	return d.createTables()
}
