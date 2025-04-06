---
title: "User Guide"
weight: 50             
---

## 5.1. Running the Server

Start the server binary with root/administrator privileges. Specify the port for the web interface (e.g., `-port 8080`) and optionally the path for the SQLite database (`-path ./data/gomesa.db`).


```Bash
# Example: Run natively, web UI on 8080, DB in ./data/
sudo mkdir -p ./data
sudo ./bin/goMESA-server -port 8080 -path ./data/gomesa.db
```

- This starts the NTP listener on UDP 123.
- This starts the Web Server on TCP port 8080 (or the specified port).
- It creates/opens the `gomesa.db` SQLite file in the `./data/` directory.

## 5.2. Accessing the Web Interface

Open your web browser and navigate to the address where the server is running, including the specified web port (e.g., `http://<server_ip>:8080` or `http://localhost:8080`).

## 5.3. Deploying Agents

1. Copy the appropriate compiled agent binary (`windows-agent.exe`, `linux-agent`, `macos-agent`) to the target machine.
2. Execute the agent with **administrator or root privileges**.
3. Once the agent successfully modifies the NTP settings and sends its first `PING` beacon, it should appear in the **Agents List** in the web interface, typically marked as "Alive".

TODO: Create a more comprehensive user guide with example actions, workflows, troubleshooting etc


[NEXT](../features/)