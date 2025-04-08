package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	// Import shared types and constants
	"goMESA/internal/common"

	// Import only the SQLite driver
	_ "github.com/mattn/go-sqlite3"
)

// Database handles all SQLite database operations
type Database struct {
	db     *sql.DB // The database connection pool
	dbPath string  // Path to the SQLite file
}

// NewDatabase creates a new SQLite database instance
// It expects the path to the database file.
func NewDatabase(dbPath string) (*Database, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path cannot be empty for SQLite")
	}
	database := &Database{
		dbPath: dbPath,
	}

	err := database.Connect()
	if err != nil {
		return nil, err // Return wrapped error from Connect
	}

	// Start a background task to periodically mark missing agents
	// Consider making the interval configurable
	go database.startMissingAgentChecker(1 * time.Minute)

	return database, nil
}

// Connect establishes a connection to the SQLite database file
func (d *Database) Connect() error {
	var err error

	// Connect to SQLite database file.
	// '_foreign_keys=on' ensures referential integrity.
	// '_journal_mode=WAL' can improve concurrency.
	// '_busy_timeout=5000' waits 5s if DB is locked.
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000", d.dbPath)
	d.db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to open/create SQLite database at %s: %w", d.dbPath, err)
	}

	// Set connection pool parameters (optional but recommended)
	d.db.SetMaxOpenConns(1) // SQLite generally performs best with a single writer
	d.db.SetMaxIdleConns(1)
	d.db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	err = d.db.Ping()
	if err != nil {
		d.db.Close() // Close if ping fails
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Ensure necessary tables exist
	err = d.createTables()
	if err != nil {
		d.db.Close() // Close if table creation fails
		return fmt.Errorf("failed to create necessary tables: %w", err)
	}

	log.Printf("SQLite database connection established at %s", d.dbPath)
	return nil
}

// Close closes the database connection pool
func (d *Database) Close() error {
	if d.db != nil {
		log.Printf("Closing SQLite database connection at %s", d.dbPath)
		return d.db.Close()
	}
	return nil
}

// GetDBPath returns the path to the database file
func (d *Database) GetDBPath() string {
	return d.dbPath
}

// createTables creates the necessary tables if they don't exist using SQLite syntax
func (d *Database) createTables() error {
	// Use TEXT for agent_id and status, INTEGER for command ID, TIMESTAMP for times
	// SQLite's TIMESTAMP affinity usually stores dates as TEXT (ISO8601) or INTEGER (Unix time)
	// Using TEXT with DEFAULT CURRENT_TIMESTAMP is generally well-supported.
	agentsTable := `
	CREATE TABLE IF NOT EXISTS agents (
		agent_id TEXT NOT NULL PRIMARY KEY,
		os TEXT,
		service TEXT,
		status TEXT NOT NULL DEFAULT 'ALIVE',
		first_seen TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
		last_seen TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
		network_adapter TEXT
	);`

	commandsTable := `
	CREATE TABLE IF NOT EXISTS commands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
		status TEXT NOT NULL DEFAULT 'SENT',
		output TEXT,
		FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
	);`

	// Execute table creation statements within a transaction for atomicity
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for table creation: %w", err)
	}
	defer tx.Rollback() // Rollback if commit fails or panics

	if _, err := tx.Exec(agentsTable); err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}

	if _, err := tx.Exec(commandsTable); err != nil {
		return fmt.Errorf("failed to create commands table: %w", err)
	}

	return tx.Commit() // Commit the transaction
}

func (d *Database) AddAgent(agentID, networkAdapter string) error {
	// Use INSERT ... ON CONFLICT ... DO UPDATE (UPSERT)
	// Only update network_adapter, status, and last_seen on conflict.
	// OS and Service remain untouched here, set them via UpdateAgentGroup or other means.
	query := `
		INSERT INTO agents (agent_id, network_adapter, status, first_seen, last_seen)
		VALUES (?, ?, ?, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
		ON CONFLICT(agent_id) DO UPDATE SET
			network_adapter=excluded.network_adapter,
			status=?, -- Always update status (e.g., mark as ALIVE)
			last_seen=excluded.last_seen
		WHERE agent_id = ?;
	`
	// Pass status twice: once for INSERT, once for UPDATE on conflict
	_, err := d.db.Exec(query, agentID, networkAdapter, common.AgentStatusAlive, common.AgentStatusAlive, agentID)
	if err != nil {
		// Log the error but allow ntp_server to continue processing packet
		log.Printf("Error in AddAgent for %s: %v", agentID, err)
		// Return the error if upstream needs to know
		// return fmt.Errorf("failed to add or update agent %s: %w", agentID, err)
		return nil // Or return nil to not disrupt ntp_server flow? Depends on desired behavior.
		// Let's return nil for now, assuming logging is sufficient.
	}
	// Logged in AddOrUpdateAgent in ntp_server for more context now (removed AddOrUpdateAgent)
	// Keep logging minimal here.
	return nil
}

// UpdateAgentStatus updates an agent's status and last_seen time
func (d *Database) UpdateAgentStatus(agentID string, status string) error {
	query := "UPDATE agents SET status = ?, last_seen = STRFTIME('%Y-%m-%d %H:%M:%f', 'now') WHERE agent_id = ?"
	result, err := d.db.Exec(query, status, agentID)
	if err != nil {
		return fmt.Errorf("failed to update status for agent %s: %w", agentID, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// This might not be an error if the agent just connected and AddAgent hasn't finished yet
		// log.Printf("Agent %s not found for status update (might be expected during initial connection)", agentID)
		return nil // Or return a specific "not found" error if needed upstream
	}
	return nil
}

// UpdateAgentGroup assigns a service/group tag to a specific agent
func (d *Database) UpdateAgentGroup(agentID, groupName string) error {
	// Assuming 'service' column is used for grouping
	query := "UPDATE agents SET service = ? WHERE agent_id = ?"
	result, err := d.db.Exec(query, groupName, agentID)
	if err != nil {
		return fmt.Errorf("failed to update group for agent %s: %w", agentID, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("agent %s not found for group update", agentID)
	}
	log.Printf("Agent %s assigned to group '%s'", agentID, groupName)
	return nil
}

// SetMissingAgents marks agents as MIA if they haven't been seen recently
// This should be called periodically (e.g., by startMissingAgentChecker)
func (d *Database) SetMissingAgents() error {
	// Use SQLite's datetime function
	// Agents unseen for 3 minutes, but only if they are currently ALIVE
	timeThreshold := "datetime('now', '-3 minutes')"
	statusMIA := common.AgentStatusMissing
	statusAlive := common.AgentStatusAlive

	query := fmt.Sprintf(
		`UPDATE agents SET status = ? WHERE last_seen < %s AND status = ?`,
		timeThreshold, // Injecting calculated threshold
	)

	result, err := d.db.Exec(query, statusMIA, statusAlive)
	if err != nil {
		log.Printf("Error marking missing agents: %v", err) // Log error but don't stop ticker
		return fmt.Errorf("failed to mark missing agents: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Marked %d agent(s) as MIA", rowsAffected)
	}
	return nil
}

// startMissingAgentChecker runs SetMissingAgents periodically
func (d *Database) startMissingAgentChecker(interval time.Duration) {
	if interval <= 0 {
		log.Println("Missing agent checker disabled (interval zero or negative)")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting missing agent checker (interval: %s)", interval)
	for range ticker.C {
		// Error is logged within SetMissingAgents
		_ = d.SetMissingAgents()
	}
	log.Println("Missing agent checker stopped.")
}

// scanAgent is a helper to scan a row into a common.Agent struct
func scanAgent(rows *sql.Rows) (common.Agent, error) {
	var agent common.Agent
	var service sql.NullString // Handle potentially NULL service
	var os sql.NullString      // Handle potentially NULL os
	var adapter sql.NullString // Handle potentially NULL adapter
	var firstSeenStr, lastSeenStr string

	err := rows.Scan(
		&agent.ID,
		&os,      // Scan into NullString
		&service, // Scan into NullString
		&agent.Status,
		&firstSeenStr,
		&lastSeenStr,
		&adapter, // Scan into NullString
	)
	if err != nil {
		return agent, err // Return error to caller
	}

	// Assign values from NullString if valid
	agent.OS = os.String
	agent.Service = service.String
	agent.NetworkAdapter = adapter.String

	// Parse timestamp strings (adjust format if needed based on createTables)
	layout := "2006-01-02 15:04:05.999" // Matches STRFTIME format used
	agent.FirstSeen, _ = time.Parse(layout, firstSeenStr)
	agent.LastSeen, _ = time.Parse(layout, lastSeenStr)
	// Ignore parsing errors for now, or handle them more robustly

	return agent, nil
}

// GetAllAgents returns all agents from the database, ordered by service then ID
func (d *Database) GetAllAgents() ([]common.Agent, error) {
	query := `
		SELECT agent_id, os, service, status, first_seen, last_seen, network_adapter
		FROM agents
		ORDER BY COALESCE(service, '') ASC, agent_id ASC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []common.Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			// Log the specific error detail but continue if possible?
			log.Printf("Error scanning agent row: %v", err)
			// Decide: return error immediately or try to return partial results?
			// Returning immediately is usually safer.
			return nil, fmt.Errorf("failed to scan agent row: %w", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agent rows: %w", err)
	}

	// Log the full agent data being returned from the database
	agentsJSON, _ := json.MarshalIndent(agents, "", "  ")
	log.Printf("DATABASE → SERVER: GetAllAgents returning: %s", string(agentsJSON))

	return agents, nil
}

// GetAgentsByGroup returns agents filtered by group type (os, service, or agent_id)
func (d *Database) GetAgentsByGroup(groupType, value string) ([]common.Agent, error) {
	var columnName string
	switch strings.ToLower(groupType) {
	case "os":
		columnName = "os"
	case "service":
		columnName = "service"
	case "agent_id":
		columnName = "agent_id"
	default:
		return nil, fmt.Errorf("invalid group type specified: %s (must be 'os', 'service', or 'agent_id')", groupType)
	}

	// Use parameterized query for the value
	query := fmt.Sprintf(
		`SELECT agent_id, os, service, status, first_seen, last_seen, network_adapter
		 FROM agents
		 WHERE %s = ?
		 ORDER BY COALESCE(service, '') ASC, agent_id ASC`,
		columnName, // Column name injection (validated above)
	)

	rows, err := d.db.Query(query, value)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents by group (%s = %s): %w", columnName, value, err)
	}
	defer rows.Close()

	var agents []common.Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			log.Printf("Error scanning grouped agent row: %v", err)
			return nil, fmt.Errorf("failed to scan grouped agent row: %w", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating grouped agent rows: %w", err)
	}

	return agents, nil
}

// RemoveAllAgents removes all agents and their commands from the database
func (d *Database) RemoveAllAgents() error {
	// Delete agents (commands should cascade delete due to ON DELETE CASCADE)
	_, err := d.db.Exec("DELETE FROM agents")
	if err != nil {
		return fmt.Errorf("failed to remove all agents: %w", err)
	}
	log.Println("Removed all agent records from the database.")
	// Optionally VACUUM to reclaim space, but can be slow
	// _, _ = d.db.Exec("VACUUM")
	return nil
}

// AddCommand adds a new command to the database for a specific agent
func (d *Database) AddCommand(agentID, content string) (int64, error) {
	query := "INSERT INTO commands (agent_id, content, status) VALUES (?, ?, ?)"
	statusSent := common.CommandStatusSent

	result, err := d.db.Exec(query, agentID, content, statusSent)
	if err != nil {
		return 0, fmt.Errorf("failed to add command for agent %s: %w", agentID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		// This might happen if the table doesn't have AUTOINCREMENT or similar issues
		// It's less common with SQLite's default INTEGER PRIMARY KEY behavior
		log.Printf("Could not retrieve last insert ID after adding command for %s: %v", agentID, err)
		return 0, fmt.Errorf("failed to get command ID after insert: %w", err)
	}

	return id, nil
}

// UpdateCommandOutput updates a command's output and sets its status to executed
func (d *Database) UpdateCommandOutput(commandID string, output string) error {
	query := "UPDATE commands SET output = ?, status = ? WHERE id = ?"
	statusExecuted := common.CommandStatusExecuted // Use constant from common

	result, err := d.db.Exec(query, output, statusExecuted, commandID)
	if err != nil {
		return fmt.Errorf("failed to update output for command %d: %w", commandID, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("Warning: Command %d not found for output update", commandID)
		// Decide if this is an error or just a warning
		// return fmt.Errorf("command %d not found for output update", commandID)
	}
	return nil
}

// GetCommandHistory returns the command history for a specific agent
func (d *Database) GetCommandHistory(agentID string) ([]common.Command, error) {
	query := `
		SELECT id, agent_id, content, timestamp, status, output
		FROM commands
		WHERE agent_id = ?
		ORDER BY timestamp DESC
	`
	rows, err := d.db.Query(query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query command history for agent %s: %w", agentID, err)
	}
	defer rows.Close()

	var commands []common.Command
	for rows.Next() {
		var cmd common.Command
		var output sql.NullString // Handle potentially NULL output
		var timestampStr string
		err := rows.Scan(
			&cmd.ID,
			&cmd.AgentID,
			&cmd.Content,
			&timestampStr,
			&cmd.Status,
			&output, // Scan into NullString
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command history row: %w", err)
		}
		if output.Valid {
			cmd.Output = output.String // Assign if not NULL
		}
		// Parse timestamp
		layout := "2006-01-02 15:04:05.999"
		cmd.Timestamp, _ = time.Parse(layout, timestampStr)

		commands = append(commands, cmd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating command history rows: %w", err)
	}

	return commands, nil
}

// CleanDatabase removes all data from tables (more robust than DROP/CREATE if schema changes)
func (d *Database) CleanDatabase() error {
	log.Println("Cleaning database (deleting all commands and agents)...")
	// Delete commands first (or rely on cascade)
	if _, err := d.db.Exec("DELETE FROM commands"); err != nil {
		return fmt.Errorf("failed to delete commands during clean: %w", err)
	}
	// Delete agents
	if _, err := d.db.Exec("DELETE FROM agents"); err != nil {
		return fmt.Errorf("failed to delete agents during clean: %w", err)
	}
	log.Println("Database cleaned.")
	// Optionally VACUUM to potentially shrink the file size
	// if _, err := d.db.Exec("VACUUM"); err != nil {
	// 	log.Printf("Warning: VACUUM failed after cleaning database: %v", err)
	// }
	return nil
}
