package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"goMESA/internal/server"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// WebSocketServer manages WebSocket connections
type WebSocketServer struct {
	db        *server.Database
	ntpServer *server.NTPServer
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
	mutex     sync.Mutex
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(db *server.Database, ntpServer *server.NTPServer) *WebSocketServer {
	return &WebSocketServer{
		db:        db,
		ntpServer: ntpServer,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
	}
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() {
	// Start the broadcast handler
	go s.handleBroadcasts()

	// Start periodic agent updates
	go s.sendPeriodicAgentUpdates()
}

// handleBroadcasts sends messages to all connected clients
func (s *WebSocketServer) handleBroadcasts() {
	for {
		message := <-s.broadcast
		s.mutex.Lock()
		clientCount := len(s.clients)
		log.Printf("SERVER BROADCAST: Sending message to %d connected clients", clientCount)
		for client := range s.clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mutex.Unlock()
	}
}

// sendPeriodicAgentUpdates sends agent updates to clients
func (s *WebSocketServer) sendPeriodicAgentUpdates() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		agents, err := s.db.GetAllAgents()
		if err != nil {
			log.Printf("Error fetching agents for update: %v", err)
			continue
		}

		// Log detailed agent count and IDs
		agentIDs := make([]string, len(agents))
		for i, agent := range agents {
			agentIDs[i] = agent.ID
		}
		log.Printf("SERVER PROCESSING: Preparing to broadcast %d agents: %v", len(agents), agentIDs)

		message := map[string]interface{}{
			"type":   "agentUpdate",
			"agents": agents,
		}

		jsonData, err := json.Marshal(message)
		// Add pretty-printing for better readability
		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, jsonData, "", "  ")
		log.Printf("SERVER → CLIENT: Broadcasting WebSocket message:\n%s", prettyJSON.String())

		s.broadcast <- jsonData
	}
}

// HandleWebSocket handles WebSocket connections
func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("CLIENT CONNECTED: New WebSocket client from %s", r.RemoteAddr)

	// Register new client
	s.mutex.Lock()
	s.clients[conn] = true
	s.mutex.Unlock()

	// Send initial agent list
	agents, err := s.db.GetAllAgents()
	if err != nil {
		log.Printf("Error fetching initial agents: %v", err)
	} else {
		message := map[string]interface{}{
			"type":   "agentUpdate",
			"agents": agents,
		}

		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling initial agent update: %v", err)
		} else {
			conn.WriteMessage(websocket.TextMessage, jsonData)
		}
	}

	// Handle incoming messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			s.mutex.Lock()
			delete(s.clients, conn)
			s.mutex.Unlock()
			break
		}

		if messageType == websocket.TextMessage {
			s.handleClientMessage(conn, p)
		}
	}
}

// handleClientMessage processes messages from clients
func (s *WebSocketServer) handleClientMessage(conn *websocket.Conn, message []byte) {
	var data map[string]interface{}
	err := json.Unmarshal(message, &data)
	if err != nil {
		log.Printf("Error parsing client message: %v", err)
		return
	}

	msgType, ok := data["type"].(string)
	if !ok {
		log.Printf("Invalid message format, missing 'type' field")
		return
	}

	switch msgType {
	case "getAgents":
		s.handleGetAgents(conn)
	case "executeCommand":
		log.Printf("SERVER RECEIVED: Command request for agent %s: '%s'",
			data["agentId"].(string), data["command"].(string))
		s.handleExecuteCommand(conn, data)
	case "pingAgent":
		s.handlePingAgent(conn, data)
	case "killAgent":
		s.handleKillAgent(conn, data)
	case "groupAgent":
		s.handleGroupAgent(conn, data)
	default:
		log.Printf("Unknown message type: %s", msgType)
	}
}

// handleGetAgents sends the list of agents to a client
func (s *WebSocketServer) handleGetAgents(conn *websocket.Conn) {
	agents, err := s.db.GetAllAgents()
	if err != nil {
		log.Printf("Error fetching agents: %v", err)
		s.sendErrorResponse(conn, "Failed to fetch agents")
		return
	}

	response := map[string]interface{}{
		"type":   "agentUpdate",
		"agents": agents,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling agent response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)
}

// handleExecuteCommand executes a command on an agent
func (s *WebSocketServer) handleExecuteCommand(conn *websocket.Conn, data map[string]interface{}) {
	agentID, ok := data["agentId"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing agentId")
		return
	}

	command, ok := data["command"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing command")
		return
	}

	// Execute the command
	output, err := s.ntpServer.ExecuteCommand(agentID, command)

	response := map[string]interface{}{
		"type":      "commandResponse",
		"agentId":   agentID,
		"command":   command,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["output"] = output
		response["success"] = true
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling command response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)

	// Also broadcast the command result to all clients
	s.broadcast <- jsonData
}

// handlePingAgent sends a ping to an agent
func (s *WebSocketServer) handlePingAgent(conn *websocket.Conn, data map[string]interface{}) {
	agentID, ok := data["agentId"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing agentId")
		return
	}

	err := s.ntpServer.SendPingCommand(agentID)

	response := map[string]interface{}{
		"type":      "pingResponse",
		"agentId":   agentID,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["success"] = true
		response["message"] = "Ping sent successfully"
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling ping response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)
}

// handleKillAgent sends a kill command to an agent
func (s *WebSocketServer) handleKillAgent(conn *websocket.Conn, data map[string]interface{}) {
	agentID, ok := data["agentId"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing agentId")
		return
	}

	err := s.ntpServer.SendKillCommand(agentID)

	response := map[string]interface{}{
		"type":      "killResponse",
		"agentId":   agentID,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["success"] = true
		response["message"] = "Kill command sent successfully"
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling kill response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)

	// Also broadcast the kill command to update all clients
	s.broadcast <- jsonData
}

// handleGroupAgent assigns a group/service to an agent
func (s *WebSocketServer) handleGroupAgent(conn *websocket.Conn, data map[string]interface{}) {
	agentID, ok := data["agentId"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing agentId")
		return
	}

	groupName, ok := data["groupName"].(string)
	if !ok {
		s.sendErrorResponse(conn, "Missing groupName")
		return
	}

	err := s.db.UpdateAgentGroup(agentID, groupName)

	response := map[string]interface{}{
		"type":      "groupResponse",
		"agentId":   agentID,
		"groupName": groupName,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["success"] = true
		response["message"] = "Agent grouped successfully"
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling group response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)

	// Trigger an agent list update
	go s.handleGetAgents(conn)
}

// sendErrorResponse sends an error response to a client
func (s *WebSocketServer) sendErrorResponse(conn *websocket.Conn, errorMsg string) {
	response := map[string]interface{}{
		"type":    "error",
		"message": errorMsg,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling error response: %v", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, jsonData)
}
