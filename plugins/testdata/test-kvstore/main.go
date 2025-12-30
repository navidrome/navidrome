// Test KVStore plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-kvstore.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	pdk "github.com/extism/go-pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
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
		_, err := host.KVStoreSet(input.Key, input.Value)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{})
		return 0

	case "get":
		resp, err := host.KVStoreGet(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Value: resp.Value, Exists: resp.Exists})
		return 0

	case "delete":
		_, err := host.KVStoreDelete(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{})
		return 0

	case "has":
		resp, err := host.KVStoreHas(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Exists: resp.Exists})
		return 0

	case "list":
		resp, err := host.KVStoreList(input.Prefix)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Keys: resp.Keys})
		return 0

	case "get_storage_used":
		resp, err := host.KVStoreGetStorageUsed()
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{StorageUsed: resp.Bytes})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
		return 0
	}
}

func main() {}
