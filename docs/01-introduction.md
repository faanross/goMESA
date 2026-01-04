# Introduction

## Overview

goMESA is an educational Command and Control (C2) framework designed to demonstrate how legitimate network protocols can be repurposed for covert communications. It uses the Network Time Protocol (NTP) as its transport mechanism, embedding command and control traffic within seemingly normal time synchronization packets.

This project is based on [mesa](https://github.com/d3adzo/mesa), with significant enhancements:

- Complete rewrite in Go with concurrency support
- SQLite database for simpler deployment
- Vue.js web interface for real-time operations
- Docker support for containerized deployment
- AES-256-GCM encryption option
- Reflective DLL loading capability
- Comprehensive documentation

## Core Components

goMESA consists of three main components:

### 1. C2 Server

The server runs on an operator-controlled system and performs dual functions:

- **Legitimate NTP Server**: Responds to standard time synchronization requests from any NTP client
- **C2 Controller**: Manages agent connections, queues commands, and stores results in a database

This dual functionality provides plausible deniability - even under inspection, the server appears to be legitimately serving time.

### 2. Agent

The agent runs on target systems (Windows, Linux, or macOS) and:

- Configures the host system to use the C2 server for time synchronization
- Uses packet capture (libpcap) to monitor NTP traffic without interfering with normal operations
- Receives commands via NTP packets and executes them using the system shell
- Returns results chunked within NTP-like packets

### 3. Web Interface

A Vue.js application provides real-time control:

- WebSocket connection for live updates
- Dashboard with agent statistics
- Command execution interface
- Agent grouping and management

## Features

| Feature | Description |
|---------|-------------|
| **NTP-Based Covert Channel** | All C2 communications use UDP port 123 |
| **Functional NTP Server** | Provides legitimate time synchronization |
| **Cross-Platform Agents** | Supports Windows, Linux, and macOS |
| **SQLite Database** | Portable, no external database required |
| **Agent Grouping** | Organize agents by OS or custom tags |
| **Dual Encryption** | XOR obfuscation and AES-256-GCM options |
| **Packet-Level Operations** | Raw packet capture for stealthy operation |
| **Reflective Loading** | In-memory DLL execution (Windows) |
| **Real-Time UI** | WebSocket-based live updates |

## Educational Value

This framework demonstrates several security concepts:

1. **Protocol abuse** - Using legitimate protocols for unintended purposes
2. **Covert channels** - Hiding communications within normal traffic patterns
3. **Evasion techniques** - Bypassing network security monitoring
4. **Detection opportunities** - Understanding what defenders should look for

By studying how such tools work, security professionals can better understand both offensive techniques and defensive countermeasures.

---

Next: [Theory & Concepts](02-theory-concepts.md)
