// Test Cache plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-cache-plugin.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

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
		err := host.CacheSetString(input.Key, input.StringVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_string":
		value, exists, err := host.CacheGetString(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{StringVal: value, Exists: exists})
		return 0

	case "set_int":
		err := host.CacheSetInt(input.Key, input.IntVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_int":
		value, exists, err := host.CacheGetInt(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{IntVal: value, Exists: exists})
		return 0

	case "set_float":
		err := host.CacheSetFloat(input.Key, input.FloatVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_float":
		value, exists, err := host.CacheGetFloat(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{FloatVal: value, Exists: exists})
		return 0

	case "set_bytes":
		err := host.CacheSetBytes(input.Key, input.BytesVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_bytes":
		value, exists, err := host.CacheGetBytes(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{BytesVal: value, Exists: exists})
		return 0

	case "has":
		exists, err := host.CacheHas(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{Exists: exists})
		return 0

	case "remove":
		err := host.CacheRemove(input.Key)
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
