// Test KVStore plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-kvstore.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// TestKVStoreInput is the input for nd_test_kvstore callback.
type TestKVStoreInput struct {
	Operation  string   `json:"operation"`             // "set", "get", "delete", "has", "list", "get_storage_used", "set_with_ttl", "delete_by_prefix", "get_many"
	Key        string   `json:"key"`                   // Storage key
	Value      []byte   `json:"value"`                 // For set operations
	Prefix     string   `json:"prefix"`                // For list/delete_by_prefix operations
	TTLSeconds int64    `json:"ttl_seconds,omitempty"` // For set_with_ttl
	Keys       []string `json:"keys,omitempty"`        // For get_many
}

// TestKVStoreOutput is the output from nd_test_kvstore callback.
type TestKVStoreOutput struct {
	Value        []byte            `json:"value,omitempty"`
	Values       map[string][]byte `json:"values,omitempty"`
	Exists       bool              `json:"exists,omitempty"`
	Keys         []string          `json:"keys,omitempty"`
	StorageUsed  int64             `json:"storage_used,omitempty"`
	DeletedCount int64             `json:"deleted_count,omitempty"`
	Error        *string           `json:"error,omitempty"`
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

	case "set_with_ttl":
		err := host.KVStoreSetWithTTL(input.Key, input.Value, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{})
		return 0

	case "delete_by_prefix":
		deletedCount, err := host.KVStoreDeleteByPrefix(input.Prefix)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{DeletedCount: deletedCount})
		return 0

	case "get_many":
		values, err := host.KVStoreGetMany(input.Keys)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestKVStoreOutput{Values: values})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestKVStoreOutput{Error: &errStr})
		return 0
	}
}

func main() {}
