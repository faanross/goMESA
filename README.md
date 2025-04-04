# goMESA

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Version](https://img.shields.io/badge/version-1.0.0-green.svg)

An educational Command & Control (C2) framework leveraging the Network Time Protocol (NTP) as a covert communication channel, featuring a modern Vue.js web interface.

![goMESA Dashboard](https://via.placeholder.com/800x450?text=goMESA+Dashboard)

## Features

- **NTP-Based Covert Channel**: Uses UDP port 123 for stealthy communication
- **Cross-Platform Agents**: Compatible with Windows, Linux, and macOS
- **Modern Web Interface**: Intuitive Vue.js dashboard for C2 operations
- **Dual-Mode Server**: Functions as both a legitimate NTP server and C2 controller
- **Database Integration**: Supports both SQLite (portable) and MySQL (scalable)
- **Real-Time Monitoring**: WebSocket-based interface with live updates
- **Agent Grouping**: Organize agents by OS type or custom service tags
- **Encryption**: XOR and AES-256-GCM encryption options for communications

## Quick Start

### Prerequisites

- Go 1.16 or higher
- Node.js 16+ and npm (for the web interface)
- Root/administrator privileges (for NTP port 123)
- Platform-specific requirements:
    - Windows: Npcap or WinPcap
    - Linux: libpcap-dev (`apt-get install libpcap-dev`)
    - macOS: libpcap (`brew install libpcap`)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/goMESA.git
   cd goMESA
   ```

2. Build the server and web interface:
   ```bash
   go build -o goMESA cmd/server/main.go
   
   cd ui
   npm install
   npm run build
   cd ..
   ```

3. Build the agents for your target platforms:
   ```bash
   GOOS=windows GOARCH=amd64 go build -o agents/windows-agent.exe cmd/agent/main.go
   GOOS=linux GOARCH=amd64 go build -o agents/linux-agent cmd/agent/main.go
   GOOS=darwin GOARCH=amd64 go build -o agents/macos-agent cmd/agent/main.go
   ```

### Running the Server

Start the server with root/administrator privileges:

```bash
sudo ./goMESA -port 8080
```

Access the web interface at `http://localhost:8080`

### Deploying Agents

1. Transfer the appropriate agent to your target system
2. Run with administrator/root privileges:
    - Windows: Right-click and "Run as Administrator"
    - Linux/macOS: `sudo ./linux-agent` or `sudo ./macos-agent`
3. The agent will connect back to your C2 server

## Usage

### Web Interface

The interface provides several key sections:

- **Dashboard**: Overview of agent statistics and recent activity
- **Agents**: Detailed view and management of connected agents
- **Commands**: Historical record of executed commands
- **Settings**: Server configuration and database management

### Key Operations

- **Execute Commands**: Run shell commands on connected agents
- **Agent Groups**: Organize agents with custom tags
- **Real-Time Updates**: Monitor agent status and command outputs
- **Database Management**: Clean or export agent data

## Architecture

goMESA consists of three main components:

1. **C2 Server**: Handles both legitimate NTP requests and agent communication
2. **Agents**: Run on target systems, communicating via NTP packets
3. **Web Interface**: Provides an intuitive control panel for operators

Communication flows through NTP packets (UDP port 123), with command and control data embedded beyond the standard NTP packet structure. Packet capture (via libpcap) allows agents to monitor NTP traffic without interfering with legitimate time services.

## Security Considerations

goMESA is designed for educational purposes and authorized security testing only:

- Use only in environments where you have explicit permission
- Be aware that NTP traffic monitoring can detect unusual patterns
- Consider implementing additional operational security measures for sensitive environments

## Docker Support

For containerized deployment:

```bash
docker build -t gomesa .
docker run -p 123:123/udp -p 8080:8080 --cap-add=NET_ADMIN gomesa
```

## Documentation

For complete documentation, visit [goMESA Docs](https://yourusername.github.io/goMESA-docs/)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This software is provided for educational and research purposes only. The author does not take responsibility for any misuse of this software. Only use goMESA in environments where you have explicit permission to conduct security testing.

## Acknowledgements

- [gopacket](https://github.com/google/gopacket) for packet capture functionality
- [Vue.js](https://vuejs.org/) for the web interface framework
- [NTP Protocol](https://tools.ietf.org/html/rfc5905) specifications