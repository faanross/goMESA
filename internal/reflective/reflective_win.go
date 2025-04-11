//go:build windows
// +build windows

package reflective

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var kernel32DLL = windows.NewLazyDLL("kernel32.dll")

// Add these structures and constants for reflective loading

// PE structures
type IMAGE_DOS_HEADER struct {
	Magic    uint16     // Magic number (MZ)
	Cblp     uint16     // Bytes on last page of file
	Cp       uint16     // Pages in file
	Crlc     uint16     // Relocations
	Cparhdr  uint16     // Size of header in paragraphs
	MinAlloc uint16     // Minimum extra paragraphs needed
	MaxAlloc uint16     // Maximum extra paragraphs needed
	Ss       uint16     // Initial (relative) SS value
	Sp       uint16     // Initial SP value
	Csum     uint16     // Checksum
	Ip       uint16     // Initial IP value
	Cs       uint16     // Initial (relative) CS value
	Lfarlc   uint16     // File address of relocation table
	Ovno     uint16     // Overlay number
	Res      [4]uint16  // Reserved words
	Oemid    uint16     // OEM identifier (for e_oeminfo)
	Oeminfo  uint16     // OEM information; e_oemid specific
	Res2     [10]uint16 // Reserved words
	Lfanew   int32      // File address of new exe header (PE header offset)
}

type IMAGE_FILE_HEADER struct {
	Machine              uint16 // Architecture type
	NumberOfSections     uint16 // Number of sections
	TimeDateStamp        uint32 // Time and date stamp
	PointerToSymbolTable uint32 // Pointer to symbol table
	NumberOfSymbols      uint32 // Number of symbols
	SizeOfOptionalHeader uint16 // Size of optional header
	Characteristics      uint16 // File characteristics
}

type IMAGE_DATA_DIRECTORY struct {
	VirtualAddress uint32 // RVA of the directory
	Size           uint32 // Size of the directory
}

type IMAGE_OPTIONAL_HEADER64 struct {
	Magic                       uint16 // Magic number (0x20b for PE32+)
	MajorLinkerVersion          uint8
	MinorLinkerVersion          uint8
	SizeOfCode                  uint32
	SizeOfInitializedData       uint32
	SizeOfUninitializedData     uint32
	AddressOfEntryPoint         uint32 // RVA of the entry point
	BaseOfCode                  uint32
	ImageBase                   uint64 // Preferred base address
	SectionAlignment            uint32
	FileAlignment               uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion           uint16
	MinorImageVersion           uint16
	MajorSubsystemVersion       uint16
	MinorSubsystemVersion       uint16
	Win32VersionValue           uint32
	SizeOfImage                 uint32 // Total size of the image in memory
	SizeOfHeaders               uint32 // Size of headers
	CheckSum                    uint32
	Subsystem                   uint16
	DllCharacteristics          uint16
	SizeOfStackReserve          uint64
	SizeOfStackCommit           uint64
	SizeOfHeapReserve           uint64
	SizeOfHeapCommit            uint64
	LoaderFlags                 uint32
	NumberOfRvaAndSizes         uint32
	DataDirectory               [16]IMAGE_DATA_DIRECTORY
}

type IMAGE_NT_HEADERS64 struct {
	Signature      uint32 // PE signature ("PE\0\0")
	FileHeader     IMAGE_FILE_HEADER
	OptionalHeader IMAGE_OPTIONAL_HEADER64
}

type IMAGE_SECTION_HEADER struct {
	Name                 [8]byte // Section name
	VirtualSize          uint32  // Actual size used in memory
	VirtualAddress       uint32  // RVA of the section
	SizeOfRawData        uint32  // Size of section data on disk
	PointerToRawData     uint32  // File offset of section data
	PointerToRelocations uint32  // File offset of relocations
	PointerToLinenumbers uint32  // File offset of line numbers
	NumberOfRelocations  uint16  // Number of relocations
	NumberOfLinenumbers  uint16  // Number of line numbers
	Characteristics      uint32  // Section characteristics
}

type IMAGE_EXPORT_DIRECTORY struct {
	Characteristics       uint32
	TimeDateStamp         uint32
	MajorVersion          uint16
	MinorVersion          uint16
	Name                  uint32 // RVA of the DLL name
	Base                  uint32 // Starting ordinal number
	NumberOfFunctions     uint32 // Total number of functions
	NumberOfNames         uint32 // Number of named functions
	AddressOfFunctions    uint32 // RVA of the EAT
	AddressOfNames        uint32 // RVA of the name pointers
	AddressOfNameOrdinals uint32 // RVA of the ordinals
}

// PE constants
const (
	IMAGE_DOS_SIGNATURE             = 0x5A4D           // "MZ"
	IMAGE_NT_SIGNATURE              = 0x00004550       // "PE\0\0"
	IMAGE_FILE_MACHINE_AMD64        = 0x8664           // x64 architecture
	IMAGE_DIRECTORY_ENTRY_EXPORT    = 0                // Export directory
	IMAGE_DIRECTORY_ENTRY_IMPORT    = 1                // Import directory
	IMAGE_DIRECTORY_ENTRY_BASERELOC = 5                // Base relocation table
	IMAGE_REL_BASED_DIR64           = 10               // 64-bit address relocation
	IMAGE_ORDINAL_FLAG64            = uintptr(1) << 63 // Import by ordinal
	DLL_PROCESS_ATTACH              = 1                // Reason for DllMain
)

// WindowsReflectiveLoader implements ReflectiveLoader for Windows
type WindowsReflectiveLoader struct{}

// NewReflectiveLoader creates a new platform-specific implementation
func NewReflectiveLoader() ReflectiveLoader {
	return &WindowsReflectiveLoader{}
}

// IsSupported always returns true on Windows
func (l *WindowsReflectiveLoader) IsSupported() bool {
	return true
}

// LoadAndExecuteDLL implements ReflectiveLoader.LoadAndExecuteDLL for Windows
func (l *WindowsReflectiveLoader) LoadAndExecuteDLL(dllBytes []byte, functionName string) (bool, error) {
	// This simply delegates to your existing implementation
	return reflectivelyLoadDLL(dllBytes, functionName)
}

// processRelocations applies base relocations if the DLL was loaded at a different address
func processRelocations(dllBase uintptr, relocDirRVA uint32, relocDirSize uint32, delta int64) {
	if delta == 0 {
		fmt.Println("    [*] Delta is zero, no relocations needed.")
		return
	}

	relocBlockAddr := dllBase + uintptr(relocDirRVA)
	maxRelocAddr := relocBlockAddr + uintptr(relocDirSize)
	totalFixups := 0

	for relocBlockAddr < maxRelocAddr {
		relocBlock := (*IMAGE_BASE_RELOCATION)(unsafe.Pointer(relocBlockAddr))

		// Check for empty or invalid block
		if relocBlock.VirtualAddress == 0 || relocBlock.SizeOfBlock == 0 {
			fmt.Println("    [*] Encountered zero block, stopping relocation processing.")
			break
		}

		// Calculate number of relocation entries in this block
		numEntries := (relocBlock.SizeOfBlock - 8) / 2
		fmt.Printf("    [*] Relocation Block: Page RVA=0x%X, Entries=%d\n",
			relocBlock.VirtualAddress, numEntries)

		// Pointer to the first relocation entry
		entryPtr := relocBlockAddr + unsafe.Sizeof(IMAGE_BASE_RELOCATION{})
		blockFixups := 0

		for i := uint32(0); i < numEntries; i++ {
			entry := *(*uint16)(unsafe.Pointer(entryPtr + uintptr(i*2)))
			relocType := entry >> 12
			relocOffset := entry & 0xFFF

			// We only care about 64-bit relocations
			if relocType == IMAGE_REL_BASED_DIR64 {
				fixAddr := dllBase + uintptr(relocBlock.VirtualAddress) + uintptr(relocOffset)
				originalValue := *(*uint64)(unsafe.Pointer(fixAddr))
				newValue := uint64(int64(originalValue) + delta)
				*(*uint64)(unsafe.Pointer(fixAddr)) = newValue
				blockFixups++
				totalFixups++
			}
		}

		// Move to the next relocation block
		relocBlockAddr += uintptr(relocBlock.SizeOfBlock)
	}

	fmt.Printf("[+] Relocations processing complete. Total fixups applied: %d\n", totalFixups)
}

// reflectivelyLoadDLL loads a DLL from memory and calls the specified function
func reflectivelyLoadDLL(dllBytes []byte, functionName string) (bool, error) {
	fmt.Println("[+] Starting reflective DLL loading...")

	dllPtr := uintptr(unsafe.Pointer(&dllBytes[0]))

	// 1. Parse PE Headers
	dosHeader := (*IMAGE_DOS_HEADER)(unsafe.Pointer(dllPtr))
	if dosHeader.Magic != IMAGE_DOS_SIGNATURE {
		return false, fmt.Errorf("invalid DOS signature")
	}

	ntHeader := (*IMAGE_NT_HEADERS64)(unsafe.Pointer(dllPtr + uintptr(dosHeader.Lfanew)))
	if ntHeader.Signature != IMAGE_NT_SIGNATURE {
		return false, fmt.Errorf("invalid PE signature")
	}

	if ntHeader.FileHeader.Machine != IMAGE_FILE_MACHINE_AMD64 {
		return false, fmt.Errorf("not a 64-bit PE file")
	}

	fmt.Printf("[+] DLL Entry Point RVA: 0x%X\n", ntHeader.OptionalHeader.AddressOfEntryPoint)
	fmt.Printf("[+] DLL Preferred Base: 0x%X\n", ntHeader.OptionalHeader.ImageBase)
	fmt.Printf("[+] DLL Image Size: 0x%X\n", ntHeader.OptionalHeader.SizeOfImage)

	// 2. Allocate Memory for DLL
	allocBase, err := windows.VirtualAlloc(
		uintptr(ntHeader.OptionalHeader.ImageBase),
		uintptr(ntHeader.OptionalHeader.SizeOfImage),
		windows.MEM_RESERVE|windows.MEM_COMMIT,
		windows.PAGE_EXECUTE_READWRITE,
	)

	if err != nil {
		fmt.Println("[*] Could not allocate at preferred base, using arbitrary address...")
		allocBase, err = windows.VirtualAlloc(
			0,
			uintptr(ntHeader.OptionalHeader.SizeOfImage),
			windows.MEM_RESERVE|windows.MEM_COMMIT,
			windows.PAGE_EXECUTE_READWRITE,
		)
		if err != nil {
			return false, fmt.Errorf("failed to allocate memory: %v", err)
		}
	}

	fmt.Printf("[+] Allocated memory at: 0x%X\n", allocBase)
	defer windows.VirtualFree(allocBase, 0, windows.MEM_RELEASE)

	// 3. Copy Headers and Sections
	copySizeHeaders := uintptr(ntHeader.OptionalHeader.SizeOfHeaders)
	var bytesWritten uintptr
	err = windows.WriteProcessMemory(windows.CurrentProcess(), allocBase, &dllBytes[0], copySizeHeaders, &bytesWritten)
	if err != nil {
		return false, fmt.Errorf("failed to copy headers: %v", err)
	}

	// 4. Copy Sections
	sectionHeaderPtr := uintptr(unsafe.Pointer(ntHeader)) + unsafe.Sizeof(*ntHeader)
	numSections := int(ntHeader.FileHeader.NumberOfSections)
	sectionHeaderSize := unsafe.Sizeof(IMAGE_SECTION_HEADER{})

	for i := 0; i < numSections; i++ {
		sectionHeader := (*IMAGE_SECTION_HEADER)(unsafe.Pointer(sectionHeaderPtr + uintptr(i)*sectionHeaderSize))
		sectionName := windows.ByteSliceToString(sectionHeader.Name[:])

		// Calculate source and destination addresses
		sectionSrc := dllPtr + uintptr(sectionHeader.PointerToRawData)
		sectionDst := allocBase + uintptr(sectionHeader.VirtualAddress)

		sizeToCopy := uintptr(sectionHeader.SizeOfRawData)
		if sizeToCopy == 0 {
			continue // Nothing to copy
		}

		fmt.Printf("    [*] Copying section %s to VA 0x%X\n", sectionName, sectionDst)

		err = windows.WriteProcessMemory(
			windows.CurrentProcess(),
			sectionDst,
			(*byte)(unsafe.Pointer(sectionSrc)),
			sizeToCopy,
			&bytesWritten,
		)
		if err != nil {
			return false, fmt.Errorf("failed to copy section %s: %v", sectionName, err)
		}
	}

	// 5. Process Relocations
	deltaImageBase := int64(allocBase) - int64(ntHeader.OptionalHeader.ImageBase)
	if deltaImageBase != 0 {
		fmt.Printf("[+] Image rebased (delta: 0x%X), processing relocations...\n", deltaImageBase)
		relocDirRVA := ntHeader.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_BASERELOC].VirtualAddress
		relocDirSize := ntHeader.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_BASERELOC].Size

		if relocDirRVA != 0 && relocDirSize != 0 {
			processRelocations(allocBase, relocDirRVA, relocDirSize, deltaImageBase)
		}
	}

	// 6. Process Import Address Table
	fmt.Println("[+] Processing Import Address Table...")
	importDirRVA := ntHeader.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_IMPORT].VirtualAddress

	if importDirRVA != 0 {
		importDescPtr := allocBase + uintptr(importDirRVA)
		importDescSize := unsafe.Sizeof(windows.ImageImportDescriptor{})

		for i := 0; ; i++ {
			importDesc := (*windows.ImageImportDescriptor)(unsafe.Pointer(importDescPtr + uintptr(i)*importDescSize))

			if importDesc.Name == 0 && importDesc.FirstThunk == 0 {
				break // End of import descriptors
			}

			dllNameRVA := importDesc.Name
			dllNamePtr := (*byte)(unsafe.Pointer(allocBase + uintptr(dllNameRVA)))
			dllName := windows.BytePtrToString(dllNamePtr)

			// Load the dependency DLL
			hModule, err := windows.LoadLibrary(dllName)
			if err != nil {
				return false, fmt.Errorf("failed to load dependency %s: %v", dllName, err)
			}

			// Determine the IAT to patch
			iatRVA := importDesc.FirstThunk
			iatBase := allocBase + uintptr(iatRVA)

			// Use ILT if available, otherwise use IAT
			iltRVA := importDesc.OriginalFirstThunk
			if iltRVA == 0 {
				iltRVA = iatRVA
			}
			iltBase := allocBase + uintptr(iltRVA)

			// Process each import
			for j := uintptr(0); ; j++ {
				iltEntryAddr := iltBase + (j * unsafe.Sizeof(uintptr(0)))
				iatEntryAddr := iatBase + (j * unsafe.Sizeof(uintptr(0)))

				iltEntry := *(*uintptr)(unsafe.Pointer(iltEntryAddr))
				if iltEntry == 0 {
					break // End of imports for this DLL
				}

				var funcAddr uintptr

				if iltEntry&IMAGE_ORDINAL_FLAG64 != 0 {
					// Import by ordinal
					ordinal := iltEntry & 0xFFFF
					getProcAddr := kernel32DLL.NewProc("GetProcAddress")
					ret, _, _ := getProcAddr.Call(uintptr(hModule), ordinal)
					funcAddr = ret
				} else {
					// Import by name
					funcNamePtr := (*byte)(unsafe.Pointer(allocBase + uintptr(iltEntry) + 2))
					funcName := windows.BytePtrToString(funcNamePtr)
					funcAddr, err = windows.GetProcAddress(hModule, funcName)
				}

				if funcAddr == 0 {
					return false, fmt.Errorf("failed to resolve import")
				}

				// Patch the IAT
				*(*uintptr)(unsafe.Pointer(iatEntryAddr)) = funcAddr
			}
		}
	}

	// 7. Find and Call the Exported Function
	exportDirRVA := ntHeader.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
	if exportDirRVA == 0 {
		return false, fmt.Errorf("DLL has no exports")
	}

	exportDir := (*IMAGE_EXPORT_DIRECTORY)(unsafe.Pointer(allocBase + uintptr(exportDirRVA)))

	// Get pointers to export tables
	eatBase := allocBase + uintptr(exportDir.AddressOfFunctions)
	enptBase := allocBase + uintptr(exportDir.AddressOfNames)
	eotBase := allocBase + uintptr(exportDir.AddressOfNameOrdinals)

	var targetFuncAddr uintptr

	// Search for the specified function
	for i := uint32(0); i < exportDir.NumberOfNames; i++ {
		nameRVA := *(*uint32)(unsafe.Pointer(enptBase + uintptr(i*4)))
		funcNamePtr := (*byte)(unsafe.Pointer(allocBase + uintptr(nameRVA)))
		funcName := windows.BytePtrToString(funcNamePtr)

		if funcName == functionName {
			ordinal := *(*uint16)(unsafe.Pointer(eotBase + uintptr(i*2)))
			funcRVA := *(*uint32)(unsafe.Pointer(eatBase + uintptr(ordinal*4)))
			targetFuncAddr = allocBase + uintptr(funcRVA)
			break
		}
	}

	if targetFuncAddr == 0 {
		return false, fmt.Errorf("function %s not found in exports", functionName)
	}

	fmt.Printf("[+] Found function %s at 0x%X\n", functionName, targetFuncAddr)

	// 8. Call DllMain first
	dllEntryRVA := ntHeader.OptionalHeader.AddressOfEntryPoint
	if dllEntryRVA != 0 {
		entryPointAddr := allocBase + uintptr(dllEntryRVA)
		ret, _, _ := syscall.SyscallN(entryPointAddr, allocBase, DLL_PROCESS_ATTACH, 0)
		if ret == 0 {
			return false, fmt.Errorf("DllMain returned FALSE")
		}
	}

	// 9. Call the target function
	ret, _, _ := syscall.SyscallN(targetFuncAddr)
	success := ret != 0

	fmt.Printf("[+] Called function %s, return value: %d\n", functionName, ret)

	return success, nil
}
