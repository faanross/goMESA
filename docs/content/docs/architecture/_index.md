---
title: "Architecture"
weight: 30             
---

## 3.1. Overview (Server & Agent)

goMESA follows a client-server model where multiple Agents connect back to a central C2 Server using NTP as the communication channel.

## 3.2. Server Component

- **Responsibilities**: Listens on UDP port 123, serves legitimate NTP requests, identifies and processes agent communications, manages agent state, queues and sends commands, receives and stores results.
- **Database**: Stores agent metadata (IP, OS, tags, status), command history, and outputs. Supports SQLite: Default, file-based, portable, suitable for smaller deployments.
- **OS Compatibility & Requirements**: Written in Go, compiles cross-platform (Linux, macOS, Windows, BSD). Requires root/administrator privileges to bind to UDP port 123. Depends on libpcap/Npcap for network operations if running agent-like functions, though primarily needs port access.

## 3.3. Agent Component

- **Responsibilities**: Establishes covert communication, mimics a legitimate NTP client, listens for commands via packet capture, executes commands using the system shell, and returns results chunked within NTP packets.
- **System Integration**: Detects host OS (Windows, Linux, macOS), identifies network configuration, modifies system NTP settings to point to the C2 server (e.g., Windows Time service, `/etc/ntp.conf`).
- **Packet Capture**: Uses libpcap/Npcap via gopacket to passively sniff for NTP packets from the C2 server destined for its IP address, avoiding conflicts with the system's network stack.
- **Cross-Platform Compatibility**: Tailored implementations for:
    - **Windows**: Integrates with Windows Time service, uses `cmd.exe`.
    - **Linux**: Modifies NTP config (e.g., `ntp.conf`, `timesyncd.conf`), uses `/bin/sh`.
    - **macOS**: Modifies macOS time service settings, uses `/bin/sh`. Requires root/administrator privileges for installation and packet capture.

## 3.4. Communication Protocol

goMESA layers its C2 protocol over standard NTP.

### 3.4.1. Packet Structure

```
+----------------+---------------+----------------+
| NTP Header     | goMESA Header | Payload        |
| (Standard 48B) | (4 bytes)     | (Variable)     |
+----------------+---------------+----------------+
```

- **NTP Header**: A valid, standard NTP header ensures the packet appears legitimate.
- **goMESA Header**: A 4-byte identifier placed in a field normally ignored or used for authentication (often within the first optional extension field or appended data not strictly part of the 48B base header, depending on implementation details hinted at). This header indicates the packet type.
- **Payload**: Encrypted command or response data.

_Note: The exact placement of the goMESA header/payload relative to the 48-byte structure can vary but aims to be ignored by standard NTP implementations._

### 3.4.2. Packet Types

The 4-byte goMESA header signifies the purpose:

- `PING`: Agent heartbeat/check-in.
- `COMU`: Command chunk (part of a larger command, Unfinished).
- `COMD`: Command chunk (final part of a command, Done).
- `COMO`: Command Output chunk.
- `KILL`: Agent termination signal.

### 3.4.3. Command Encoding & Chunking

Commands and large outputs are broken into smaller chunks to fit within reasonable packet sizes. Each chunk is sent in a separate NTP-like packet, marked with the appropriate header (`COMU`, `COMD`, `COMO`). The receiving end reassembles these chunks based on sequence and the final `COMD` marker.

### 3.4.4. Encryption (XOR, AES-256-GCM)

Payloads are encrypted for obfuscation and security:

- **XOR**: Simple, fast obfuscation using a fixed key. Easily reversible if the key is known but prevents casual inspection.
- **AES-256-GCM**: Stronger, authenticated encryption. Requires a shared key and proper nonce management to prevent replay attacks.

### 3.4.5. Handshaking

The initial `PING` packet sent by a new agent acts as a registration/handshake, allowing the server to add the agent to its database. Subsequent `PING`s serve as heartbeats.


## 3.5. Communication Flow Summary

1. **Agent -> Server (NTP)**: Heartbeats (`PING`), Command Outputs (`COMO`).
2. **Server -> Agent (NTP)**: Commands (`COMU`/`COMD`), Kill signals (`KILL`).
3. **Web Client <-> Server (WebSocket/HTTP)**:
  - Client sends requests (view agents, execute command, ping, kill, group) via WebSocket messages or REST API calls.
  - Server pushes real-time updates (agent list, status changes, command results) to clients via WebSocket.
  - Server serves initial web application files via HTTP.


[NEXT](../setup/)