---
title: "Theory and Concepts"
weight: 20             
---

## 2.1. NTP as a Covert Channel

NTP as transport protocol leverages several characteristics that make it interesting for covert communication.

### 2.1.1. Why NTP Works

1. **Ubiquity & Necessity**: NTP is a fundamental internet protocol required by nearly all systems for time synchronization. It's essential for logging, security mechanisms (like Kerberos), financial transactions, and more.
2. **Default Firewall Allowance**: Due to its necessity, NTP traffic (UDP port 123) is almost universally permitted through firewalls.
3. **Limited Inspection**: Unlike HTTP/S or DNS, NTP traffic is rarely subjected to deep packet inspection or significant scrutiny by security appliances.
4. **UDP Protocol**: Being UDP-based, NTP is connectionless, simplifying the process of crafting and injecting custom packets without session establishment overhead.
5. **Regular Timing Patterns**: Legitimate NTP clients communicate periodically, making the regular beaconing of C2 agents appear less suspicious than protocols with constant connections.
6. **Bidirectional Nature**: The standard request-response pattern of NTP naturally facilitates two-way C2 communication.

### 2.1.2. Legitimate Cover & Plausible Deniability

A key aspect of goMESA's design is that the C2 server functions as a _real_ NTP server. It correctly processes and responds to time requests from standard NTP clients. This dual functionality provides plausible deniability; even if the server's traffic is inspected, it appears to be legitimately serving time. The C2 communications are hidden within what looks like standard protocol interactions.

## 2.2. Background on NTP

Understanding NTP's normal operation highlights how goMESA exploits it.

### 2.2.1. The Foundation of Internet Timekeeping

Developed by David Mills in 1985, NTP synchronizes clocks across computer networks. It uses a hierarchical system of "strata," where Stratum 0 are high-precision sources (atomic clocks, GPS), Stratum 1 servers sync directly to Stratum 0, and so on. Most devices sync to servers several strata removed.

### 2.2.2. Protocol Characteristics (UDP, Port 123)

NTP operates at the Application Layer but uses UDP (typically port 123) as its transport protocol. UDP's connectionless nature is suitable for time synchronization where timeliness is valued over guaranteed delivery.

### 2.2.3. Packet Structure & Timestamps

A standard NTP packet is 48 bytes long. Key fields include:

- Leap Indicator, Version Number, Mode (Client/Server/etc.)
- Stratum level
- Poll Interval, Precision
- Root Delay, Root Dispersion
- Reference ID
- Four 64-bit timestamps (Reference, Originate, Receive, Transmit) used for calculating clock offset and round-trip delay.

![NTP packet](packet.png)

### 2.2.4. Operational Patterns

Clients poll servers periodically. The interval is adaptive, often starting frequently (e.g., 64s) and increasing (e.g., 1024s) as the clock stabilizes. Clients use the four timestamps in the request/response cycle to calculate offset and delay, gradually "slewing" their clock to match the server's time, avoiding disruptive jumps.

## 2.3. Raw Network Access Concepts (libpcap/gopacket)

goMESA agents use raw network access for stealth and control, typically via libraries like libpcap (or its Windows equivalent Npcap) accessed through Go bindings like gopacket.

### 2.3.1. Introduction to Packet Capture

Standard applications use high-level socket APIs. Packet capture libraries bypass these, tapping directly into the network interface data stream (typically at the Data Link Layer). This provides access to complete, unmodified network packets, including all headers.

### 2.3.2. Technical Foundation (Architecture, Filtering)

libpcap uses kernel-level mechanisms (e.g., BPF, AF_PACKET) to intercept packet copies. Crucially, it allows efficient kernel-level filtering (using BPF syntax like `"udp and port 123"`) so only relevant packets are passed to the user-space application, minimizing overhead. Captured packets are buffered in the kernel before being delivered to the application via the libpcap API.

### 2.3.3. Benefits and Limitations

- **Benefits**: Complete visibility, protocol independence, passive observation, custom packet crafting (injection), cross-platform API.
- **Limitations**: Requires root/admin privileges, performance overhead (kernel-user copy), complexity (protocol knowledge needed), potential for packet loss under heavy load, security implications of raw access.

### 2.3.4. Comparison to Standard Socket Programming

When a typical application uses sockets (like TCP or UDP sockets), it interacts primarily at the Transport Layer (L4) or slightly above. The operating system's networking stack handles the processing for Layers 2 (Data Link), 3 (Network), and 4 (Transport). It strips away the headers from these lower layers and delivers only the application-level payload (Layer 7 data) to the application via the socket interface.
Packet capture bypasses the operating system's standard processing path that would normally handle layers 2, 3, and 4 and deliver only the L7 payload to an application via a socket. Instead, it gives the capturing application the raw data much lower down the stack, including all the headers and the L7 payload, leaving the application responsible for parsing and interpreting all of it.

Using standard sockets for goMESA would be problematic:

- **Conflict**: Binding to UDP port 123 would conflict with the system's legitimate NTP service.
- **Indiscriminate Reception**: Would receive _all_ NTP traffic, requiring user-space filtering.
- **Interference**: Would likely disrupt normal time synchronization.
- **Limited Access**: Header information (like source IP from the IP layer) might be inaccessible or processed by the OS stack.

Using libpcap allows the goMESA agent to:

- **Coexist**: Passively monitor traffic without binding the port.
- **Precise Filtering**: Capture only packets from the C2 server destined for the agent's IP.
- **Full Packet Access**: Inspect all headers for verification and custom data extraction.
- **Stealth**: Operate alongside the legitimate NTP client.

[NEXT](../architecture/)