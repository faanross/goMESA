---
title: "Features"
weight: 3
---
        
## Features

- **NTP-Based Covert Channel**: Uses UDP port 123 for all C2 communications.
- **Functional NTP Server**: The C2 server provides legitimate time synchronization, enhancing plausible deniability.
- **Cross-Platform Agents**: Supports Windows, Linux, and macOS target systems.
- **Database Backend**: Supports SQLite (default, portable).
- **Agent Grouping**: Manage agents efficiently by OS or custom service tags.
- **Encryption**: Offers basic XOR obfuscation and stronger AES-256-GCM encryption for payloads.
- **Packet-Level Operations**: Utilizes raw packet capture (libpcap/gopacket) for stealthy agent operation.
- **Containerization Support**: Docker and Docker Compose files available for easier deployment.


