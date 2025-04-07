.PHONY: all build clean server agent-linux agent-windows agent-macos agents run

# Build variables
BINARY_DIR=bin
SERVER_NAME=mesa-server
AGENT_LINUX=linux-agent
AGENT_WINDOWS=windows-agent.exe
AGENT_MACOS=macos-agent

# Set server IP for agents during compilation - change this to your C2 server's IP
SERVER_IP=192.168.2.124

all: clean build

build: server agents

server:
	@echo "Building server..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(SERVER_NAME) ./cmd/server

agents: agent-linux agent-windows agent-macos

agent-linux:
	@echo "Building Linux agent..."
	@mkdir -p $(BINARY_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.serverIP=$(SERVER_IP)" -o $(BINARY_DIR)/$(AGENT_LINUX) ./cmd/agent

agent-windows:
	@echo "Building Windows agent..."
	@mkdir -p $(BINARY_DIR)
	@GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.serverIP=$(SERVER_IP)" -o $(BINARY_DIR)/$(AGENT_WINDOWS) ./cmd/agent

agent-macos:
	@echo "Building macOS agent..."
	@mkdir -p $(BINARY_DIR)
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.serverIP=$(SERVER_IP)" -o $(BINARY_DIR)/$(AGENT_MACOS) ./cmd/agent

run: server
	@echo "Starting Mesa C2 server..."
	@sudo ./$(BINARY_DIR)/$(SERVER_NAME)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)