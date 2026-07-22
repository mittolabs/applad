package storage

import (
	"fmt"
	"net"
	"time"
)

// ScanResult holds the result of an antivirus scan.
type ScanResult struct {
	Clean   bool
	Message string
}

// ScanWithClamAV scans file content using ClamAV's clamd TCP interface.
// Expects clamd to be listening on the given address (e.g., "clamav:3310").
func ScanWithClamAV(addr string, content []byte) (*ScanResult, error) {
	if addr == "" {
		return &ScanResult{Clean: true, Message: "antivirus not configured"}, nil
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("antivirus: connect to clamd: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Send INSTREAM command
	_, err = conn.Write([]byte("zINSTREAM\x00"))
	if err != nil {
		return nil, fmt.Errorf("antivirus: send command: %w", err)
	}

	// Send content in chunks with 4-byte big-endian length prefix
	chunkSize := 2048
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunk := content[i:end]
		size := uint32(len(chunk))
		header := []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}
		conn.Write(header)
		conn.Write(chunk)
	}

	// Send zero-length chunk to signal end
	conn.Write([]byte{0, 0, 0, 0})

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("antivirus: read response: %w", err)
	}

	response := string(buf[:n])

	// ClamAV response format: "stream: OK" or "stream: <virus> FOUND"
	if len(response) > 0 && response[len(response)-1] == 0 {
		response = response[:len(response)-1]
	}

	if response == "stream: OK" {
		return &ScanResult{Clean: true, Message: "clean"}, nil
	}

	return &ScanResult{Clean: false, Message: response}, nil
}
