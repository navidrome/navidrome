// Test Library plugin for Navidrome plugin system integration tests.
// This plugin tests library metadata access WITH filesystem permission,
// allowing tests for both metadata and filesystem access.
// Build with: tinygo build -o ../test-library.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"os"
	"path/filepath"

	pdk "github.com/extism/go-pdk"
)

// TestLibraryInput is the input for nd_test_library callback.
type TestLibraryInput struct {
	Operation  string `json:"operation"` // "get_library", "get_all_libraries", "read_file", "list_dir"
	LibraryID  int32  `json:"library_id,omitempty"`
	MountPoint string `json:"mount_point,omitempty"` // For filesystem operations
	FilePath   string `json:"file_path,omitempty"`   // For read_file operation (relative to mount point)
}

// TestLibraryOutput is the output from nd_test_library callback.
type TestLibraryOutput struct {
	Library     *Library  `json:"library,omitempty"`
	Libraries   []Library `json:"libraries,omitempty"`
	FileContent string    `json:"file_content,omitempty"`
	DirEntries  []string  `json:"dir_entries,omitempty"`
	Error       *string   `json:"error,omitempty"`
}

// nd_test_library is the test callback that tests the library host functions.
//
//go:wasmexport nd_test_library
func ndTestLibrary() int32 {
	var input TestLibraryInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "get_library":
		resp, err := LibraryGetLibrary(input.LibraryID)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestLibraryOutput{Library: resp.Result})
		return 0

	case "get_all_libraries":
		resp, err := LibraryGetAllLibraries()
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestLibraryOutput{Libraries: resp.Result})
		return 0

	case "read_file":
		// Read a file from the mounted library directory
		fullPath := filepath.Join(input.MountPoint, input.FilePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestLibraryOutput{FileContent: string(content)})
		return 0

	case "list_dir":
		// List files in the mounted library directory
		entries, err := os.ReadDir(input.MountPoint)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
			return 0
		}
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		pdk.OutputJSON(TestLibraryOutput{DirEntries: names})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestLibraryOutput{Error: &errStr})
		return 0
	}
}

func main() {}
