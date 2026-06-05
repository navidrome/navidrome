// Test Config plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-config.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// TestConfigInput is the input for nd_test_config callback.
type TestConfigInput struct {
	Operation string `json:"operation"` // "get", "get_int", "list"
	Key       string `json:"key"`       // For get/get_int operations
	Prefix    string `json:"prefix"`    // For list operation
}

// TestConfigOutput is the output from nd_test_config callback.
type TestConfigOutput struct {
	StringVal string   `json:"string_val,omitempty"`
	IntVal    int64    `json:"int_val,omitempty"`
	Keys      []string `json:"keys,omitempty"`
	Exists    bool     `json:"exists,omitempty"`
	Error     *string  `json:"error,omitempty"`
}

// nd_test_config is the test callback that tests the config host functions.
//
//go:wasmexport nd_test_config
func ndTestConfig() int32 {
	var input TestConfigInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestConfigOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "get":
		value, exists := host.ConfigGet(input.Key)
		pdk.OutputJSON(TestConfigOutput{StringVal: value, Exists: exists})
		return 0

	case "get_int":
		value, exists := host.ConfigGetInt(input.Key)
		pdk.OutputJSON(TestConfigOutput{IntVal: value, Exists: exists})
		return 0

	case "list":
		keys := host.ConfigKeys(input.Prefix)
		pdk.OutputJSON(TestConfigOutput{Keys: keys})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestConfigOutput{Error: &errStr})
		return 0
	}
}

func main() {}
