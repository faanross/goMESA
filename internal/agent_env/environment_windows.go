//go:build windows
// +build windows

package agent_env // or whatever package this belongs in

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

// getEnvironmentalID generates a unique client ID based on system information
func GetEnvironmentalID() (string, error) {
	// Windows-specific implementation using GetVolumeInformation
	var volumeName [256]uint16
	var volumeSerial uint32

	err := windows.GetVolumeInformation(
		windows.StringToUTF16Ptr("C:\\"),
		&volumeName[0],
		uint32(len(volumeName)),
		&volumeSerial,
		nil,
		nil,
		nil,
		0,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get volume info: %v", err)
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %v", err)
	}

	// Combine information
	shortName := hostname
	if len(hostname) > 5 {
		shortName = hostname[:5]
	}

	clientID := fmt.Sprintf("%s-%x", shortName, volumeSerial)
	return clientID, nil
}
