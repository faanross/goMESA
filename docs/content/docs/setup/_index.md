---
title: "Setup and Installation"
weight: 40             
---

## 4.1. Prerequisites

- **Go**: Version 1.16 or higher (for building server/agent).
- **Node.js & npm**: Version 16+ (for building the Vue.js web interface).
- **Packet Capture Library**:
    - Linux: `libpcap-dev`
    - macOS: `libpcap` (usually pre-installed or via `brew`)
    - Windows: Npcap (recommended)
- **Root/Administrator Privileges**: Required to run the server (bind port 123) and agents (install, packet capture, NTP config).
- **Build Tools**: `make` (if using a Makefile), standard Go build tools, npm.


## 4.2. Building the Framework

Building goMESA involves compiling the Go backend (server and agents) and the Vue.js frontend. You can choose between using the provided `Makefile`  or running manual `go build` for the Go components. Building the frontend requires Node.js + npm regardless of the backend build method.


### 4.2.1. Clone the Repository

```Bash 
git clone https://github.com/faanross/goMESA.git
cd goMESA
```


### 4.2.2. Configure Server IP for Agents:

The agent binaries need to know the IP address of the C2 server to connect back to. This IP is embedded during compilation. You must configure this before building the agents, using the method corresponding to your chosen build approach:

#### 4.2.2.1. Makefile

```makefile
# In Makefile - Edit this line
SERVER_IP=YOUR_SERVER_IP_ADDRESS
```


#### 4.2.2.2. go build
- Manually add the `-ldflags "-X main.serverIP=YOUR_SERVER_IP"` flag to each `go build` command for the agents


### 4.2.3. Building Server + Agent

#### 4.2.3.1. Makefile

The `Makefile` automates the Go compilation process.

* **Ensure you completed Step 2 for the Makefile.**
* Run `make` commands from the project's root directory:
    * `make build`: Builds the server and all agent binaries.
    * `make all`: Cleans previous builds (`make clean`) and then runs `make build`.
    * `make server`: Builds only the server binary (`./bin/mesa-server`).
    * `make agents`: Builds all agent binaries (`./bin/linux-agent`, `./bin/windows-agent.exe`, `./bin/macos-agent`).
    * `make agent-linux` / `make agent-windows` / `make agent-macos`: Builds only the specific agent.
    * `make clean`: Removes the `bin/` directory and compiled artifacts.
    * `make run`: Builds the server (if needed) and runs it using `sudo`.

* **Output:** Binaries are placed in the `./bin/` directory.

#### 4.2.3.2. go build

* **Create Output Directory:**
    ```bash
    mkdir -p bin
    ```
* **Build the Server:**
    ```bash
    go build -ldflags="-s -w" -o ./bin/mesa-server ./cmd/server/main.go
    ```
* **Build the Agents:**
    * **Linux Agent:**
        ```bash
        GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" -o ./bin/linux-agent_env ./cmd/agent_env/agent_env.go
        ```
    * **Windows Agent:**
        ```bash
        GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" -o ./bin/windows-agent_env.exe ./cmd/agent_env/agent_env.go
        ```
    * **macOS Agent:**
        ```bash
        GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.serverIP=YOUR_SERVER_IP" -o ./bin/macos-agent_env ./cmd/agent_env/agent_env.go
        ```


### 4.2.4. Building Client UI

```Bash
cd ui          # Navigate into the ui directory
npm install    # Install Node.js dependencies
npm run build  # Compile the Vue.js application
cd ..          # Return to the project root directory
```

- **Output:** This creates the static web application files (HTML, CSS, JS) needed by the server in the `./ui/dist/` directory.


### 4.2.5. Completion:

After following these steps (choosing either Makefile or Manual go build for the Go components, and completing the UI build), you will have the necessary compiled artifacts ready for deployment:

- Server binary (e.g., `./bin/mesa-server`)
- Agent binaries with embedded server IP (e.g., `./bin/*-agent*`)
- Compiled web UI files (`./ui/dist/`)


## 4.3. Database Setup (SQLite)

- No manual database setup is required.
- When the server starts, it will automatically create the SQLite database file (e.g., `goMESA.db`) in its working directory or at the path specified using the `-path` command-line flag, if the file doesn't already exist.
- Ensure the directory where the server runs (or the specified path) has write permissions for the user running the server process.

[NEXT](../guide/)
