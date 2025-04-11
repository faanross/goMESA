package api

import (
	"encoding/json"
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
}

// NewAPIServer creates a new API server
func NewAPIServer(db *server.Database, ntpServer *server.NTPServer, port int) *APIServer {
	router := mux.NewRouter()

	wsServer := NewWebSocketServer(db, ntpServer)

	// Create HTTPS server
	httpsServer := https_server.NewHTTPSServer(
		"server.crt",  // Default TLS certificate path
		"server.key",  // Default TLS private key path
		"0.0.0.0:443", // Default HTTPS port
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
	}

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

	// Start the HTTPS server
	if err := s.httpsServer.Start(); err != nil {
		log.Printf("Warning: Failed to start HTTPS server: %v", err)
		// Continue anyway, as this is not critical
	}
	log.Println("Starting HTTPS server.")

	log.Printf("API Server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the API server
func (s *APIServer) Stop() error {
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

// handleGetAgent returns a specific agent
func (s *APIServer) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	agents, err := s.db.GetAgentsByGroup("agent_id", agentID)
	if err != nil {
		http.Error(w, "Failed to get agent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(agents) == 0 {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents[0])
}

// handleGetAgentCommands returns commands for a specific agent
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

// handlePingAgent sends a ping to an agent
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

// handleKillAgent sends a kill command to an agent
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

// handleGroupAgent assigns a group/service to an agent
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
