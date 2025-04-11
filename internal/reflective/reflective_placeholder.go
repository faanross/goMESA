// just a placeholder implement lin/darwin specific builds later if desired

//go:build !windows
// +build !windows

package reflective

import (
	"fmt"
)

// OtherPlatformLoader implements ReflectiveLoader for non-Windows platforms
type OtherPlatformLoader struct{}

// NewReflectiveLoader creates a new platform-specific implementation
func NewReflectiveLoader() ReflectiveLoader {
	return &OtherPlatformLoader{}
}

// LoadAndExecuteDLL implements ReflectiveLoader.LoadAndExecuteDLL for non-Windows
func (l *OtherPlatformLoader) LoadAndExecuteDLL(dllBytes []byte, functionName string) (bool, error) {
	return false, fmt.Errorf("reflective loading not supported on this platform")
}

// IsSupported always returns false on non-Windows platforms
func (l *OtherPlatformLoader) IsSupported() bool {
	return false
}
