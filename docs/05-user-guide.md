# User Guide

This section covers day-to-day operations: running the server, deploying agents, and using the web interface.

## Running the Server

### Starting the Server

Start with root/administrator privileges:

```bash
# Basic startup
sudo ./bin/mesa-server

# With custom options
sudo ./bin/mesa-server -port 8080 -path ./data/gomesa.db

# With reflective loading support
sudo ./bin/mesa-server -port 8080 -server-ip YOUR_EXTERNAL_IP
```

The server will:

1. Start the NTP listener on UDP port 123
2. Start the web server on the specified port (default 8080)
3. Start the HTTPS server on port 443 (for payload delivery)
4. Create/open the SQLite database

### Verifying Server Status

Check the console output for:

```
Starting NTP server on :123
Starting HTTPS server on 0.0.0.0:443
API Server starting on :8080
```

Access the web interface at `http://localhost:8080` (or your server's IP).

## Deploying Agents

### Preparation

1. Build the agent for your target platform (see [Setup](04-setup.md))
2. Transfer the agent binary to the target system
3. Ensure the target has the required packet capture library installed

### Execution

Run the agent with elevated privileges:

**Windows:**
- Right-click the executable and select "Run as Administrator"
- Or from an elevated command prompt: `windows-agent.exe`

**Linux/macOS:**
```bash
sudo ./linux-agent
# or
sudo ./macos-agent
```

### What Happens on Execution

1. Agent modifies NTP settings to use the C2 server
2. Agent begins packet capture on the network interface
3. Agent sends initial PING (registration beacon)
4. Agent appears in the web interface within seconds

### Verification

- Check the web interface's Agents list
- New agents appear with status "Alive"
- Agent information shows IP, OS, and first-seen timestamp

## Web Interface

### Dashboard

The main dashboard provides an overview:

- **Agent Count**: Total registered agents
- **Active Agents**: Currently responding agents
- **Recent Activity**: Latest commands and responses

### Agents View

Lists all registered agents with:

| Column | Description |
|--------|-------------|
| ID | Unique agent identifier |
| IP | Agent's network address |
| OS | Operating system |
| Status | Alive, MIA, or Killed |
| Last Seen | Timestamp of last heartbeat |
| First Seen | Initial registration time |
| Group | Custom tag/service name |

**Agent States:**
- **Alive**: Responding to heartbeats
- **MIA**: No response for 60+ seconds
- **Killed**: Terminated by operator

### Executing Commands

1. Select an agent from the list
2. Enter a command in the execute panel
3. Click "Execute" or press Enter
4. Wait for output to appear

Commands are queued and delivered via the next NTP exchange. Response time depends on the agent's polling interval.

**Example Commands:**

```bash
# Windows
whoami
ipconfig /all
Get-Process

# Linux/macOS
id
ifconfig
ps aux
```

### Agent Actions

| Action | Effect |
|--------|--------|
| **Ping** | Send immediate heartbeat request |
| **Kill** | Terminate the agent process |
| **Group** | Assign a custom tag/service name |

### Command History

The Commands view shows:

- All executed commands
- Target agent
- Timestamp
- Output/results
- Execution status

## Operational Workflow

### Typical Session

1. **Start Server**
   ```bash
   sudo ./bin/mesa-server -port 8080 -path ./data/gomesa.db
   ```

2. **Open Web Interface**
   - Navigate to `http://server-ip:8080`

3. **Deploy Agents**
   - Transfer and execute agent on targets
   - Verify appearance in web interface

4. **Conduct Operations**
   - Select agents
   - Execute commands
   - Review output
   - Group agents for organization

5. **Cleanup**
   - Send KILL commands to agents
   - Stop server with Ctrl+C

### Grouping Agents

Use groups to organize agents by purpose:

- `webservers` - Web server hosts
- `domain-controllers` - AD infrastructure
- `workstations` - End-user systems

Groups help manage operations across related systems.

## Detection Considerations

For threat hunting and blue team exercises, be aware of these detection opportunities:

### Network Indicators

- NTP traffic to non-standard time servers
- Unusually frequent NTP requests
- NTP packets larger than standard 48 bytes
- NTP responses without corresponding requests

### Host Indicators

- Modified NTP configuration files
- New NTP server entries in Windows Time service
- Processes with raw socket/packet capture capabilities
- Unexpected network connections on UDP 123

### Logging

- Monitor NTP client configuration changes
- Track processes binding to network interfaces
- Log firewall rule modifications
- Audit administrative actions

## Troubleshooting

### Agent Not Appearing

1. **Network connectivity**: Can target reach server on UDP 123?
2. **Privileges**: Is agent running as admin/root?
3. **Packet capture**: Is libpcap/Npcap installed and working?
4. **Firewall**: Is UDP 123 allowed outbound?

### Commands Not Executing

1. **Agent status**: Is agent showing as "Alive"?
2. **Queue time**: Commands queue until next NTP exchange
3. **Output size**: Large outputs take multiple packets

### Agent Shows MIA

- Agent may have been terminated
- Network connectivity lost
- Target system shutdown
- Firewall blocked return traffic

### Server Won't Start

1. **Port conflict**: Is another service using port 123?
2. **Privileges**: Running as root/administrator?
3. **Database**: Is the data directory writable?

---

[Back to Documentation Index](README.md)
