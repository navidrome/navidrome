// Test plugin for Storage host function integration tests.
// Build with: tinygo build -o ../test-subsonicapi-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

//go:wasmexport call_read
func callRead() int32 {
	path := filepath.Join(host.StorageGetStoragePath(), pdk.InputString())
	content, err := os.ReadFile(path)
	if err != nil {
		pdk.SetErrorString("failed to read file: " + err.Error())
		return 1
	}
	pdk.Output(content)
	return 0
}

type WriteInput struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

//go:wasmexport call_write
func callWrite() int32 {
	var config WriteInput
	err := pdk.InputJSON(&config)

	if err != nil {
		pdk.SetErrorString("failed to parse json: " + err.Error())
		return 1
	}

	path := filepath.Join(host.StorageGetStoragePath(), config.Path)
	err = os.WriteFile(path, []byte(config.Contents), 0600)
	if err != nil {
		pdk.SetErrorString("failed to write file: " + err.Error())
		return 1
	}

	return 0
}

func main() {}
