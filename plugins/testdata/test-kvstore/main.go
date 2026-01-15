// Test KVStore plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-kvstore.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// TestKVStoreInput is the input for nd_test_kvstore callback.
type TestKVStoreInput struct {
	Operation string `json:"operation"` // "set", "get", "delete", "has", "list", "get_storage_used"
	Key       string `json:"key"`       // Storage key
	Value     []byte `json:"value"`     // For set operations
	Prefix    string `json:"prefix"`    // For list operation
}

// TestKVStoreOutput is the output from nd_test_kvstore callback.
type TestKVStoreOutput struct {
	Value       []byte   `json:"value,omitempty"`
	Exists      bool     `json:"exists,omitempty"`
	Keys        []string `json:"keys,omitempty"`
	StorageUsed int64    `json:"storage_used,omitempty"`
	Error       *string  `json:"error,omitempty"`
}

// nd_test_kvstore is the test callback that tests the kvstore host functions.
//
//go:wasmexport nd_test_kvstore
func ndTestKVStore() int32 {
	var input TestKVStoreInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "set":
		err := host.KVStoreSet(input.Key, input.Value)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{})
		return 0

	case "get":
		value, exists, err := host.KVStoreGet(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Value: value, Exists: exists})
		return 0

	case "delete":
		err := host.KVStoreDelete(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{})
		return 0

	case "has":
		exists, err := host.KVStoreHas(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Exists: exists})
		return 0

	case "list":
		keys, err := host.KVStoreList(input.Prefix)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Keys: keys})
		return 0

	case "get_storage_used":
		bytesUsed, err := host.KVStoreGetStorageUsed()
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{StorageUsed: bytesUsed})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
		return 0
	}
}

func main() {}
