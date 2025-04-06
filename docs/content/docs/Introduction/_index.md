---
title: "Introduction"
weight: 10             
---

## 1.1. Overview

goMESA is an educational Command and Control (C2) framework designed to demonstrate how legitimate network protocols can be repurposed for covert communications. Specifically, it utilizes the Network Time Protocol (NTP) as its transport mechanism, embedding command and control traffic within seemingly normal time synchronization packets.

It is based on [mesa](https://github.com/d3adzo/mesa), with the following notable changes:
- Entire application is rewritten in Golang + integration of concurrency
- SQLite instead of MySQL as a simpler option
- Vue.js Web Client UI
- Docker support
- AES-256-GCM as a more secure encryption method
- Comprehensive documentation

## 1.2. Core Components

goMESA consists of three main components:

1. **C2 Server**: Runs on an operator-controlled system. It acts as a fully functional NTP server (responding to legitimate time requests) while also listening for and managing connections from goMESA agents. It provides an interface for the operator to interact with agents and stores agent information and command history in a database.
2. **Agent**: Runs on target systems (Windows, Linux, macOS). It mimics a standard NTP client, configuring the host system to use the C2 server for time synchronization. Covertly, it uses this NTP communication channel to receive commands, execute them, and send back results.
3. **Client UI**: A Vue.js client establishes a persistent WebSocket connection to the goMESA server (or a dedicated web API layer), allowing for real-time, bidirectional communication.

By operating under the guise of essential NTP traffic, goMESA aims to evade detection by network security monitoring tools that may not closely inspect UDP port 123 traffic.

## 1.3. Features

- **NTP-Based Covert Channel**: Uses UDP port 123 for all C2 communications.
- **Functional NTP Server**: The C2 server provides legitimate time synchronization, enhancing plausible deniability.
- **Cross-Platform Agents**: Supports Windows, Linux, and macOS target systems.
- **Database Backend**: Supports SQLite (default, portable).
- **Agent Grouping**: Manage agents efficiently by OS or custom service tags.
- **Encryption**: Offers basic XOR obfuscation and stronger AES-256-GCM encryption for payloads.
- **Packet-Level Operations**: Utilizes raw packet capture (libpcap/gopacket) for stealthy agent operation.
- **Built-in Persistence**: Provides OS-specific methods for agents to automatically restart and maintain C2 communication after reboots.

