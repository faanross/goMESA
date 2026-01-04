# goMESA

![gomesa](./docs/images/gomesa.png)


An educational Command & Control (C2) framework that demonstrates covert communication techniques using the Network Time Protocol (NTP).

## What is goMESA?

goMESA is a learning tool designed to illustrate how legitimate network protocols can be repurposed for covert communications. It embeds C2 traffic within standard NTP packets, making the communication appear as normal time synchronization.

**Key Concept**: The C2 server functions as a *real* NTP server, correctly responding to legitimate time requests while simultaneously managing agent communications. This dual functionality demonstrates the challenge defenders face when attackers abuse trusted protocols.

This project is based on [mesa](https://github.com/d3adzo/mesa) by d3adzo, reimplemented in Go with significant enhancements.

## Features

| Category | Features |
|----------|----------|
| **Communication** | NTP-based covert channel (UDP 123), dual-mode server (legitimate NTP + C2) |
| **Agents** | Cross-platform (Windows, Linux, macOS), packet-level operations via libpcap |
| **Encryption** | XOR obfuscation, AES-256-GCM |
| **Evasion** | Reflective DLL loading - in-memory execution without disk writes (Windows) |
| **Interface** | Vue.js web UI, real-time WebSocket updates, agent grouping |
| **Storage** | SQLite database, automatic schema creation, command history |

## Architecture

```
┌─────────────────┐                              ┌─────────────────┐
│   C2 Server     │◄──── NTP (UDP 123) ─────────►│     Agent       │
│                 │      Commands / Output       │   (Target)      │
├─────────────────┤                              └─────────────────┘
│ • NTP Server    │
│ • SQLite DB     │                              ┌─────────────────┐
│ • REST API      │◄──── NTP (UDP 123) ─────────►│     Agent       │
│ • WebSocket     │                              │   (Target)      │
└────────┬────────┘                              └─────────────────┘
         │
         │ HTTP/WS
         ▼
┌─────────────────┐
│   Web Client    │
│   (Vue.js)      │
└─────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.16+
- Node.js 16+ and npm
- Packet capture library:
  - Linux: `sudo apt install libpcap-dev`
  - macOS: libpcap (pre-installed)
  - Windows: [Npcap](https://npcap.com)
- Root/administrator privileges

### Build

```bash
# Clone repository
git clone https://github.com/faanross/goMESA.git
cd goMESA

# Configure your server IP in Makefile
# Edit: SERVER_IP=YOUR_IP_HERE

# Build everything
make build

# Build web interface
cd ui && npm install && npm run build && cd ..
```

### Run Server

```bash
sudo ./bin/mesa-server -port 8080
```

Access the web interface at `http://localhost:8080`

### Deploy Agent

Transfer the appropriate agent binary to a target system and run with elevated privileges:

```bash
# Linux/macOS
sudo ./linux-agent

# Windows (run as Administrator)
windows-agent.exe
```

The agent will appear in the web interface within seconds.

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

| Document | Description |
|----------|-------------|
| [Introduction](docs/01-introduction.md) | Project overview, features, and components |
| [Theory & Concepts](docs/02-theory-concepts.md) | NTP covert channels, packet capture principles |
| [Architecture](docs/03-architecture.md) | System design, protocol details, packet structures |
| [Setup & Installation](docs/04-setup.md) | Prerequisites, building, configuration |
| [User Guide](docs/05-user-guide.md) | Operations, web interface, troubleshooting |

## Educational Value

This framework demonstrates several security concepts valuable for both offensive and defensive practitioners:

### For Red Teams / Pentesters
- Protocol abuse techniques
- Covert channel implementation
- Evasion of network monitoring
- In-memory payload execution

### For Blue Teams / Threat Hunters
- Detection opportunities for protocol abuse
- Network indicators of compromise
- Host-based detection strategies
- Understanding attacker techniques

### Detection Indicators

Network-level:
- NTP traffic to non-standard servers
- Unusual NTP packet sizes (>48 bytes)
- High-frequency NTP requests

Host-level:
- Modified NTP configuration
- Processes with packet capture capabilities
- Unexpected UDP 123 connections

## Project Structure

```
goMESA/
├── cmd/
│   ├── agent/           # Agent entry point
│   └── server/          # Server entry point
├── internal/
│   ├── common/          # Shared utilities (crypto, models, packets)
│   ├── server/          # Server implementation (NTP, database, API)
│   └── reflective/      # Reflective loader (Windows)
├── ui/                  # Vue.js web interface
├── docs/                # Documentation
├── certs/               # TLS certificates
├── data/                # SQLite database (created at runtime)
└── Makefile             # Build automation
```

## Security Notice

**This software is provided for educational and authorized security testing purposes only.**

- Only use in environments where you have explicit permission
- Designed for controlled lab environments and security research
- Not intended for unauthorized access to computer systems
- Users are responsible for compliance with applicable laws

## Acknowledgments

- [mesa](https://github.com/d3adzo/mesa) - Original concept and implementation
- [gopacket](https://github.com/google/gopacket) - Packet capture library
- [Vue.js](https://vuejs.org/) - Web interface framework
- [Gorilla Toolkit](https://www.gorillatoolkit.org/) - HTTP routing and WebSocket

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
