# Setup and Installation

This section covers prerequisites, building the framework, and configuration.

## Prerequisites

### Required Software

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.16+ | Building server and agents |
| Node.js | 16+ | Building web interface |
| npm | (with Node.js) | Package management |
| make | (optional) | Build automation |

### Packet Capture Libraries

| Platform | Package | Installation |
|----------|---------|--------------|
| Linux | libpcap-dev | `apt install libpcap-dev` |
| macOS | libpcap | Pre-installed (or `brew install libpcap`) |
| Windows | Npcap | Download from [npcap.com](https://npcap.com) |

### Privileges

Both server and agents require elevated privileges:

- **Server**: Root/admin to bind UDP port 123
- **Agents**: Root/admin for packet capture and NTP configuration

## Building the Framework

### 1. Clone the Repository

```bash
git clone https://github.com/faanross/goMESA.git
cd goMESA
```

### 2. Configure Server IP

Agents need to know the C2 server's IP address. This is embedded at compile time.

**Using Makefile:**

Edit the `Makefile` and set your server IP:

```makefile
# IMPORTANT: Set this to your C2 server's IP address
SERVER_IP=YOUR_SERVER_IP_HERE
```

**Using manual build:**

Include the IP in the `-ldflags` parameter (shown below).

### 3. Build Options

#### Option A: Using Makefile (Recommended)

```bash
# Build everything (server + all agents)
make build

# Or build components separately:
make server          # Server only
make agents          # All agents
make agent-linux     # Linux agent only
make agent-windows   # Windows agent only
make agent-macos     # macOS agent only

# Clean build artifacts
make clean
```

Output binaries are placed in `./bin/`.

#### Option B: Manual Build

```bash
# Create output directory
mkdir -p bin

# Build server
go build -ldflags="-s -w" -o ./bin/mesa-server ./cmd/server

# Build agents (replace YOUR_SERVER_IP)
# Linux
GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" \
  -o ./bin/linux-agent ./cmd/agent

# Windows
GOOS=windows GOARCH=amd64 go build \
  -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" \
  -o ./bin/windows-agent.exe ./cmd/agent

# macOS
GOOS=darwin GOARCH=amd64 go build \
  -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" \
  -o ./bin/macos-agent ./cmd/agent
```

### 4. Build Web Interface

```bash
cd ui
npm install
npm run build
cd ..
```

This creates static files in `./ui/dist/` that the server will serve.

### 5. Verify Build

After building, you should have:

```
bin/
├── mesa-server        # C2 server
├── linux-agent        # Linux agent
├── windows-agent.exe  # Windows agent
└── macos-agent        # macOS agent

ui/dist/               # Compiled web interface
```

## Configuration

### Server Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | 8080 | Web interface port |
| `-path` | ../data/gomesa.db | SQLite database path |
| `-server-ip` | (none) | External IP for payload delivery |

Example:

```bash
sudo ./bin/mesa-server -port 8080 -path ./data/gomesa.db -server-ip 192.168.1.100
```

### Database Setup

No manual setup required. The server automatically:

- Creates the SQLite database file if it doesn't exist
- Initializes required tables
- Uses WAL mode for better concurrency

Ensure the database directory is writable by the server process.

### TLS Certificates

For HTTPS payload delivery, place certificates in the `certs/` directory:

```
certs/
├── server.crt    # TLS certificate
└── server.key    # Private key
```

Generate self-signed certificates for testing:

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
  -out certs/server.crt -days 365 -nodes \
  -subj "/CN=localhost"
```

## Docker Deployment (Optional)

Build and run with Docker:

```bash
# Build image
docker build -t gomesa .

# Run container
docker run -p 123:123/udp -p 8080:8080 -p 443:443 \
  --cap-add=NET_ADMIN gomesa
```

The `NET_ADMIN` capability is required for raw socket operations.

## Troubleshooting

### "Permission denied" on port 123

The server must run as root/administrator to bind privileged ports:

```bash
sudo ./bin/mesa-server
```

### libpcap not found (Linux)

Install the development package:

```bash
# Debian/Ubuntu
sudo apt install libpcap-dev

# RHEL/CentOS
sudo yum install libpcap-devel
```

### Agent not connecting

1. Verify the server IP was correctly embedded during build
2. Check firewall allows UDP 123 traffic
3. Ensure agent has admin/root privileges
4. Verify target can reach server on port 123

### Web interface not loading

1. Confirm `ui/dist/` exists (run `npm run build` in `ui/`)
2. Check you're accessing the correct port
3. Verify no firewall blocking the web port

---

Next: [User Guide](05-user-guide.md)
