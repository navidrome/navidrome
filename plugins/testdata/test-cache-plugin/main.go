// Test Cache plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-cache-plugin.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"

	pdk "github.com/extism/go-pdk"
)

// Manifest types
type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	Cache *CachePermission `json:"cache,omitempty"`
}

type CachePermission struct {
	Reason string `json:"reason,omitempty"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Test Cache Plugin",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A test cache plugin for integration testing",
		Permissions: &Permissions{
			Cache: &CachePermission{
				Reason: "For testing cache operations",
			},
		},
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

// TestCacheInput is the input for nd_test_cache callback.
type TestCacheInput struct {
	Operation  string  `json:"operation"`   // "set_string", "get_string", "set_int", "get_int", "set_float", "get_float", "set_bytes", "get_bytes", "has", "remove"
	Key        string  `json:"key"`         // Cache key
	StringVal  string  `json:"string_val"`  // For string operations
	IntVal     int64   `json:"int_val"`     // For int operations
	FloatVal   float64 `json:"float_val"`   // For float operations
	BytesVal   []byte  `json:"bytes_val"`   // For bytes operations
	TTLSeconds int64   `json:"ttl_seconds"` // TTL in seconds
}

// TestCacheOutput is the output from nd_test_cache callback.
type TestCacheOutput struct {
	StringVal string  `json:"string_val,omitempty"`
	IntVal    int64   `json:"int_val,omitempty"`
	FloatVal  float64 `json:"float_val,omitempty"`
	BytesVal  []byte  `json:"bytes_val,omitempty"`
	Exists    bool    `json:"exists,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// nd_test_cache is the test callback that tests the cache host functions.
//
//go:wasmexport nd_test_cache
func ndTestCache() int32 {
	var input TestCacheInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestCacheOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "set_string":
		_, err := CacheSetString(input.Key, input.StringVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_string":
		resp, err := CacheGetString(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{StringVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_int":
		_, err := CacheSetInt(input.Key, input.IntVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_int":
		resp, err := CacheGetInt(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{IntVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_float":
		_, err := CacheSetFloat(input.Key, input.FloatVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_float":
		resp, err := CacheGetFloat(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{FloatVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_bytes":
		_, err := CacheSetBytes(input.Key, input.BytesVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_bytes":
		resp, err := CacheGetBytes(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{BytesVal: resp.Value, Exists: resp.Exists})
		return 0

	case "has":
		resp, err := CacheHas(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{Exists: resp.Exists})
		return 0

	case "remove":
		_, err := CacheRemove(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestCacheOutput{Error: &errStr})
		return 0
	}
}

func main() {}
