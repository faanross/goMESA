# Architecture

This section details goMESA's system design, communication protocol, and data flow.

## System Overview

goMESA follows a client-server architecture where multiple agents connect to a central C2 server using NTP as the communication channel.

```
┌─────────────────┐         NTP (UDP 123)         ┌─────────────────┐
│                 │◄────────────────────────────► │                 │
│   C2 Server     │         Commands/Output       │     Agent       │
│                 │                               │   (Target)      │
├─────────────────┤                               └─────────────────┘
│  NTP Server     │
│  Database       │         NTP (UDP 123)         ┌─────────────────┐
│  API Server     │◄────────────────────────────► │     Agent       │
│  HTTPS Server   │                               │   (Target)      │
└────────┬────────┘                               └─────────────────┘
         │
         │ HTTP/WebSocket
         ▼
┌─────────────────┐
│   Web Client    │
│   (Vue.js)      │
└─────────────────┘
```

## Server Component

### Responsibilities

- Listen on UDP port 123 for NTP traffic
- Serve legitimate NTP requests (dual functionality)
- Identify and process agent communications
- Manage agent state and command queues
- Store data in SQLite database
- Provide REST API and WebSocket interface
- Serve HTTPS for payload delivery

### Database Schema

The SQLite database stores:

| Table | Purpose |
|-------|---------|
| `agents` | Agent metadata (ID, IP, OS, status, timestamps) |
| `commands` | Command history and outputs |

### Platform Requirements

- Written in Go (cross-platform)
- Requires root/admin to bind port 123
- Dependencies: libpcap for packet operations

## Agent Component

### Responsibilities

- Establish covert communication channel
- Configure host to use C2 server for NTP
- Monitor NTP traffic via packet capture
- Execute received commands
- Return results in chunked packets

### System Integration

The agent integrates with the target OS:

| Platform | NTP Configuration | Shell | Packet Capture |
|----------|-------------------|-------|----------------|
| Windows | Windows Time Service | cmd.exe | Npcap |
| Linux | ntp.conf / timesyncd | /bin/sh | libpcap |
| macOS | systemsetup | /bin/sh | libpcap |

### Operation Flow

1. Agent starts with elevated privileges
2. Modifies system NTP settings to point to C2 server
3. Begins packet capture with BPF filter
4. Sends initial PING (registration)
5. Monitors for incoming commands
6. Executes commands, returns chunked output
7. Sends periodic heartbeats

## Communication Protocol

goMESA layers its C2 protocol over standard NTP.

### Packet Structure

```
┌──────────────────┬───────────────┬────────────────────┐
│   NTP Header     │ goMESA Header │     Payload        │
│   (48 bytes)     │  (4 bytes)    │    (variable)      │
│   Standard NTP   │  Identifier   │  Encrypted data    │
└──────────────────┴───────────────┴────────────────────┘
```

- **NTP Header**: Valid standard header for protocol compliance
- **goMESA Header**: 4-byte type identifier
- **Payload**: XOR or AES encrypted command/output data

### Packet Types

The 4-byte header identifies the packet purpose:

| Type | Value | Direction | Purpose |
|------|-------|-----------|---------|
| PING | `PING` | Agent → Server | Heartbeat/registration |
| COMU | `COMU` | Server → Agent | Command chunk (continued) |
| COMD | `COMD` | Server → Agent | Command chunk (final) |
| COMO | `COMO` | Agent → Server | Output chunk |
| KILL | `KILL` | Server → Agent | Terminate agent |
| WCON | `WCON` | Server → Agent | Web connect (reflective load) |

### Command Chunking

Large commands and outputs are split into chunks:

```
Command: "Get-Process | Format-Table -AutoSize"

Packet 1: [NTP][COMU][Get-Process | Fo...]
Packet 2: [NTP][COMU][rmat-Table -Auto...]
Packet 3: [NTP][COMD][Size]  ← Final chunk
```

The receiver reassembles chunks until receiving the final marker (COMD for commands, implicit for output).

### Encryption Options

| Method | Use Case | Key |
|--------|----------|-----|
| XOR | Basic obfuscation | Single byte (`.`) |
| AES-256-GCM | Secure transmission | 32-byte shared key |

XOR provides minimal obfuscation against casual inspection. AES-GCM provides authenticated encryption for sensitive environments.

### Handshaking

Initial agent registration:

1. Agent sends PING packet with system information
2. Server extracts agent ID, IP, OS details
3. Server adds agent to database
4. Agent appears in web interface

Subsequent PINGs serve as heartbeats. Agents not seen for 60+ seconds are marked as "MIA".

## Communication Flow

### Agent ↔ Server (NTP)

```
Agent                                   Server
  │                                       │
  │──────── PING (registration) ─────────►│
  │                                       │ Store in DB
  │◄─────── COMU/COMD (command) ──────────│
  │                                       │
  │ Execute command                       │
  │                                       │
  │──────── COMO (output chunks) ────────►│
  │                                       │ Store output
  │                                       │
  │──────── PING (heartbeat) ────────────►│
  │                                       │ Update last_seen
```

### Web Client ↔ Server

```
Web Client                              Server
  │                                       │
  │◄════════ WebSocket Connect ══════════►│
  │                                       │
  │◄──────── Agent Updates ───────────────│ (every 5s)
  │                                       │
  │──────── Execute Command ─────────────►│
  │                                       │ Queue for agent
  │◄──────── Command Result ──────────────│
  │                                       │
  │──────── Kill Agent ──────────────────►│
  │                                       │ Send KILL packet
```

## Reflective Loading (Advanced)

For Windows targets, goMESA supports reflective DLL loading:

1. Operator uploads DLL via web interface
2. Server registers payload, generates unique URL
3. Server sends WCON packet to agent with URL
4. Agent downloads obfuscated DLL over HTTPS
5. Agent deobfuscates using derived key
6. Agent loads DLL directly into memory (no disk write)
7. Agent executes specified export function

This technique evades file-based detection by never writing the payload to disk.

---

Next: [Setup & Installation](04-setup.md)
