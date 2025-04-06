---
title: "Core Components"
weight: 2
---
        
## Core Components

goMESA consists of three main components:

1. **C2 Server**: Runs on an operator-controlled system. It acts as a fully functional NTP server (responding to legitimate time requests) while also listening for and managing connections from goMESA agents. It provides an interface for the operator to interact with agents and stores agent information and command history in a database.
2. **Agent**: Runs on target systems (Windows, Linux, macOS). It mimics a standard NTP client, configuring the host system to use the C2 server for time synchronization. Covertly, it uses this NTP communication channel to receive commands, execute them, and send back results.
3. **Client UI**: A Vue.js client establishes a persistent WebSocket connection to the goMESA server (or a dedicated web API layer), allowing for real-time, bidirectional communication.

By operating under the guise of essential NTP traffic, goMESA aims to evade detection by network security monitoring tools that may not closely inspect UDP port 123 traffic.

[NEXT PAGE](../features/)


