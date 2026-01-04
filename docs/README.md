# goMESA Documentation

Welcome to the goMESA documentation. This guide covers the theory, architecture, setup, and usage of the goMESA educational C2 framework.

## Table of Contents

1. **[Introduction](01-introduction.md)** - Project overview, features, and core components
2. **[Theory & Concepts](02-theory-concepts.md)** - NTP covert channels, packet capture, and underlying principles
3. **[Architecture](03-architecture.md)** - System design, communication protocol, and packet structures
4. **[Setup & Installation](04-setup.md)** - Prerequisites, building, and configuration
5. **[User Guide](05-user-guide.md)** - Running the server, deploying agents, and operations

## Quick Links

- [Building the Framework](04-setup.md#building-the-framework)
- [Running the Server](05-user-guide.md#running-the-server)
- [Deploying Agents](05-user-guide.md#deploying-agents)

## About This Project

goMESA is designed for educational purposes to demonstrate how legitimate network protocols can be repurposed for covert communications. It uses NTP (Network Time Protocol) as a transport mechanism, embedding C2 traffic within normal time synchronization packets.

**Important**: This framework is intended for authorized security research, threat hunting simulations, and educational environments only.
