# goMESA: NTP-Based Command & Control (C2) Framework

goMESA is a Command and Control (C2) framework that utilizes the Network Time Protocol (NTP) as its transport mechanism. 
It is inspired by [Mesa](https://github.com/d3adzo/mesa), the original "C2 over NTP" tool, but I've fully rewritten it in Go, added a bubbleTea TUI, as well as added a few extra features described below.

## Features

- **NTP-Based Communication**: Uses NTP (UDP port 123) as a covert channel for C2 communications
- **Cross-Platform Agents**: Works on Windows, Linux, and macOS
- **Legitimate NTP Server**: Functions as a real NTP time server, providing cover for C2 traffic
- **Interactive TUI**: Intuitive terminal user interface 
- **Database Storage**: Supports both SQLite (portable) and MySQL backends
- **Agent Grouping**: Group agents by OS or custom service tags
- **XOR and AES Encryption**: Provides basic payload encryption

## Architecture

Mesa consists of two primary components:

1. **C2 Server**: Written in Go, the server component listens for NTP traffic, handles agent communication, and provides an interactive interface for operators.

2. **Agents**: Cross-platform implants that establish communication with the C2 server using NTP packets, execute commands, and return results.

## Prerequisites

- Go 1.16 or higher
- For required Go packages see `go.mod` + install automatically via `go get`


- For agent functionality:
    - Windows: No additional dependencies
    - Linux: libpcap-dev (`apt-get install libpcap-dev`)
    - macOS: libpcap (`brew install libpcap`)

## Building

1. Clone the repository:
   ```
   git clone https://github.com/faanross/goMESA.git
   cd goMESA
   ```

2. Build the server and agents:
   ```
   make
   ```

   This will create the following binaries in the `bin` directory:
    - `mesa-server`: The C2 server
    - `linux-agent`: Agent for Linux systems
    - `windows-agent.exe`: Agent for Windows systems
    - `macos-agent`: Agent for macOS systems

3. Customize the server IP for agents:

   Edit the `Makefile` and change the `SERVER_IP` variable to your C2 server's IP address:
   ```
   SERVER_IP=your.server.ip.address
   ```

## Usage

### Starting the Server

Run the server with:

```
sudo ./bin/mesa-server
```

By default, the server uses SQLite for data storage. To use MySQL:

```
sudo ./bin/mesa-server -db mysql -user yourusername
```

### Deploying Agents

Transfer the appropriate agent binary to the target system and execute it with administrator/root privileges:

- **Windows**:
  ```
  windows-agent.exe
  ```

- **Linux**:
  ```
  sudo ./linux-agent
  ```

- **macOS**:
  ```
  sudo ./macos-agent
  ```

### Server Commands

The server features a multi-level command interface:

#### Main Prompt

- `agents`: Display connected agents
- `db`: Enter the database management prompt
- `interact <type> <id>`: Interact with agents (types: a[gent], o[s], s[ervice])
- `help`: Display help information
- `exit`: Exit the program, saving the state
- `shutdown`: Exit, kill all agents, and clean the database

#### DB Prompt

- `agents`: List all agent entries
- `group <ip> <os/service> <name>`: Add identifiers to agents (supports IP ranges)
- `removeall`: Remove all agents from the database
- `meta`: Describe the database schema
- `help`: Display help information
- `back`: Return to the main prompt

#### Interact Prompt

- `agents`: Display agents under the current filter
- `ping`: Ping selected agent(s)
- `kill`: Send kill command to agent(s)
- `cmd`: Enter the command prompt
- `help`: Display help information
- `back`: Return to the main prompt

#### Command Prompt

- `<command>`: Send the specified command to the agent(s)
- `help`: Display help information
- `back`: Return to the interact prompt

## Security Considerations

This tool is intended for educational and authorized security testing purposes only. Be aware that:

1. NTP traffic is not typically inspected by firewalls, which is what makes it an effective covert channel.
2. The framework performs actual NTP server functions, providing plausible deniability.
3. The current implementation includes basic encryption, but may not be sufficient for highly secure environments.

## Legal Disclaimer

This software is provided for educational and research purposes only. The author does not take responsibility for any misuse of this software. Only use Mesa in environments where you have explicit permission to conduct security testing.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

# Background Information

## The Foundation of Internet Timekeeping

The Network Time Protocol (NTP) stands as one of the oldest Internet protocols still in active use today, originally designed by David Mills at the University of Delaware in 1985. NTP serves a crucial yet often overlooked function in our interconnected digital world: it synchronizes time across computer systems. This synchronization is far more important than many realize, as precise timekeeping underpins everything from financial transactions and database operations to security mechanisms and log analysis. Without NTP, the internet as we know it would struggle to maintain the temporal coherence necessary for its myriad services and applications.

At its core, NTP addresses a fundamental challenge in distributed computing—the lack of a common clock. Computer systems each have their own internal clocks that tend to drift over time due to variations in hardware, temperature fluctuations, and other environmental factors. Even high-quality clocks can drift by several milliseconds per day, which quickly compounds into significant discrepancies across systems. NTP creates a hierarchical system where computers synchronize their time with more accurate reference clocks in a cascading fashion. This hierarchy, known as the "stratum" system, begins with highly precise atomic clocks or GPS receivers (Stratum 0), flows to directly connected time servers (Stratum 1), and then propagates throughout the network to end devices that might be several strata removed from the authoritative time sources.

## Protocol Characteristics and Technical Implementation

Within the OSI network model, NTP operates primarily at the application layer (Layer 7), though it relies on the transport layer (Layer 4) services of the User Datagram Protocol (UDP). NTP typically uses UDP port 123 for all its communications, employing the connectionless nature of UDP to minimize overhead while exchanging time information. This choice of UDP over TCP makes sense for time synchronization, as the protocol values timeliness over guaranteed delivery—a delayed time synchronization packet holds little value in a protocol designed to maintain precise timing.

The structure of an NTP packet reveals the elegant design of the protocol. Each standard NTP packet consists of a 48-byte structure without any additional payload. The first byte contains three key fields: a 2-bit leap indicator (warning of upcoming leap seconds), a 3-bit version number, and a 3-bit mode indicator that defines the role of the sender (client, server, broadcast, etc.). The subsequent bytes contain the stratum level (1-16, with lower numbers indicating more authoritative time sources), poll interval (the logarithmic value indicating frequency of checks), precision (again in logarithmic format), and various timing-related values like root delay and root dispersion, which indicate the total round-trip delay to the reference clock and the maximum error in the time estimate, respectively.

The heart of an NTP packet lies in its timestamp fields. The protocol uses 64-bit timestamps, with the first 32 bits representing seconds since January 1, 1900 (the NTP epoch), and the remaining 32 bits representing fractional seconds, providing theoretical precision to about 233 picoseconds. The packet contains four such timestamps:
- the reference timestamp (when the local clock was last updated),
- originate timestamp (when the client sent its request),
- receive timestamp (when the server received the request),
- and transmit timestamp (when the server sent its response).

These timestamps enable sophisticated algorithms that can determine network latency and clock offset with remarkable precision, allowing systems to gradually adjust their clocks to match authoritative time sources while accounting for network delays.

## Operational Patterns and Typical Implementation

In standard operation, NTP clients interact with servers through a predictable pattern. A client first establishes which NTP servers it will use, either through manual configuration or dynamic discovery protocols. Most operating systems ship with default NTP servers pre-configured, often pointing to public NTP pools maintained by volunteers around the world. The client then initiates synchronization by sending a request packet to its configured server, containing the client's current time in the originate timestamp field. The server responds with a packet that includes when it received the request (receive timestamp) and when it sent its response (transmit timestamp).

Upon receiving the server's response, the client now possesses all four timestamps needed to calculate two critical values: the round-trip delay between client and server, and the offset between the client's clock and the server's clock. The round-trip delay is calculated as (T4 - T1) - (T3 - T2), where T1 is the originate timestamp, T2 is the receive timestamp, T3 is the transmit timestamp, and T4 is the time when the client received the response. The clock offset is calculated as ((T2 - T1) + (T3 - T4))/2. This clever mathematical approach allows the client to determine how far its clock deviates from the server's without knowing the actual network latency in either direction.

Rather than immediately jumping to the new time, which could disrupt running applications, NTP typically implements a gradual slewing approach. The client will slightly speed up or slow down its clock until it converges with the correct time. This adjustment process might involve either adjusting the frequency at which the system counts clock ticks or adding/subtracting small values from the system time at regular intervals. For larger discrepancies (typically over 1000 seconds), NTP might perform a step adjustment, immediately changing the system time, but this is generally avoided during normal operation.

The polling interval—how frequently a client checks with its time servers—follows an adaptive algorithm. Initial synchronization might involve polling every 64 seconds (2^6), but this interval typically extends to longer periods as the clock stabilizes, often settling at intervals of 1024 seconds (2^10) or even longer. This adaptive approach balances network traffic against synchronization accuracy, with the protocol automatically adjusting based on observed network conditions and clock stability.

## Mesa C2 Framework: Weaponizing Time Synchronization

The Mesa Command and Control (C2) framework represents an innovative exploitation of the NTP protocol for covert communications. While conventional network monitoring often scrutinizes HTTP, HTTPS, DNS, and other common protocols for suspicious activity, NTP traffic tends to receive far less attention. This relative invisibility makes NTP an ideal candidate for a covert channel—a method of communication that exploits an existing, legitimate protocol to transmit unauthorized data. Mesa leverages this oversight in network defense to establish persistent communication channels between compromised systems (agents) and a controlling server, all under the guise of innocent time synchronization.

The genius of Mesa's approach lies in how it maintains legitimate NTP functionality while simultaneously enabling command and control capabilities. The C2 server functions as a fully operational NTP server, correctly responding to standard time requests from regular clients. This dual functionality provides excellent cover for its covert operations—even if network traffic is captured and analyzed, the packets appear to be serving a legitimate purpose. The server identifies Mesa agents through subtle markers in what would otherwise look like normal NTP communications.

Mesa's implementation extends the standard NTP packet without alerting protocol analyzers. The standard 48-byte NTP packet structure remains intact, but Mesa appends additional data beyond this structure. This approach works because most NTP implementations only process the first 48 bytes of a packet and ignore any additional data. By placing its command and control data after the standard NTP structure, Mesa ensures that legitimate NTP servers and clients will simply disregard the extra information while Mesa components recognize and process it.

The agent component of Mesa establishes its covert presence by first integrating with the target system's legitimate time services. On Windows systems, it modifies the Windows Time service configuration; on Linux and macOS, it alters the appropriate NTP configuration files. This integration serves two purposes: it provides a legitimate reason for NTP traffic to exist on the network, and it ensures the agent's traffic blends with normal system operations. Once established, the agent begins sending regular heartbeat signals to the C2 server, disguised as normal NTP synchronization requests.

These heartbeats follow the natural timing patterns of NTP clients, starting with relatively frequent checks (around once per minute) and potentially adjusting to less frequent intervals to further evade detection. Each heartbeat provides the C2 server with confirmation that the agent remains operational and receptive to commands. When the server receives a heartbeat, it can optionally embed commands in its response. These commands are encrypted using either simple XOR obfuscation or more robust AES encryption, depending on the operational security requirements.

For command transmission, Mesa employs a clever chunking mechanism to handle commands of arbitrary length. Since NTP packets are relatively small, large commands must be split across multiple packets. The server marks these command chunks with special identifiers: "COMU" (Command Unfinished) for intermediate chunks and "COMD" (Command Done) for the final chunk. The agent accumulates these chunks until it receives the final one, then reassembles and executes the complete command using the appropriate system shell for the host operating system.

Command output follows a similar pattern back to the server. The agent captures the command's standard output and standard error streams, encrypts this data, and chunks it if necessary. Each chunk is embedded in what appears to be a normal NTP client synchronization request, with special identifiers marking the chunks as command output rather than heartbeats. The C2 server reassembles these chunks to obtain the complete command output.

Unlike many C2 frameworks that rely on polling (where agents regularly check for new commands), Mesa implements a more sophisticated approach. The server maintains a queue of pending commands for each agent and delivers these commands in response to the agent's natural heartbeat signals. This approach minimizes unnecessary network traffic and better mimics legitimate NTP behavior, where clients initiate requests and servers respond, rather than servers pushing unsolicited data to clients.

To further enhance its stealth, Mesa's agent employs raw packet capture at the network interface level using libraries like libpcap. Rather than registering as a normal network application that receives packets through the operating system's standard socket interfaces, the agent directly monitors network traffic at a lower level. This approach allows it to intercept and process NTP packets destined for its host without relying on higher-level networking APIs that might be monitored for suspicious activity. The agent specifically filters for UDP packets on port 123 coming from the C2 server's IP address, ignoring all other traffic.

The Mesa framework also implements sophisticated agent management capabilities. Agents can be grouped by operating system type, assigned custom service tags, or addressed individually. This organization allows operators to efficiently manage large deployments, sending commands to specific groups of agents as needed. All agent information, command history, and outputs are stored in either a SQLite or MySQL database, providing persistence across server restarts and facilitating complex operations.

In the event that an operation needs to conclude, Mesa includes a clean termination mechanism. The server can send a special "KILL" command to agents, which then perform cleanup operations appropriate to their host operating system. On Windows, this might involve stopping and unregistering the Windows Time service; on Linux or macOS, it could include restoring original NTP configuration files. After cleanup, the agents terminate, leaving minimal traces of their presence.

The entire communication cycle of Mesa exemplifies how seemingly innocuous protocols can be repurposed for covert operations. By maintaining the external appearance of legitimate NTP traffic while encoding command and control information within that traffic, Mesa achieves a high degree of stealth. Network defenders typically focus on inspecting HTTP, HTTPS, DNS, and other common protocols for signs of compromise, while NTP traffic passes with minimal scrutiny. Even if specific NTP packets are captured, they appear to be performing their expected function of time synchronization, with the command and control data hidden in plain sight.

This exploitation of NTP represents a broader security challenge: as defenders harden traditional communication channels, attackers shift to less scrutinized protocols. The Mesa framework demonstrates why comprehensive network monitoring should extend beyond the obvious protocols to include services like NTP that, while essential for normal operations, can also serve as vectors for covert communication. Understanding these techniques helps security professionals develop more effective detection mechanisms and highlights the importance of applying the principle of least privilege even to seemingly benign services like time synchronization.
