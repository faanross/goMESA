package reflective

// ReflectiveLoader interface defines the operations for reflective loading
type ReflectiveLoader interface {
	// LoadAndExecuteDLL loads a DLL from memory and executes the specified function
	LoadAndExecuteDLL(dllBytes []byte, functionName string) (bool, error)

	// IsSupported returns true if reflective loading is supported on this platform
	IsSupported() bool
}

// PayloadMetadata represents additional data embedded with a payload
type PayloadMetadata struct {
	FunctionName string `json:"function_name"` // Name of the function to execute in the DLL
}
