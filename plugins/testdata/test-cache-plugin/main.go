// Test Cache plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-cache-plugin.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	pdk "github.com/extism/go-pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
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
		_, err := host.CacheSetString(input.Key, input.StringVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_string":
		resp, err := host.CacheGetString(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{StringVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_int":
		_, err := host.CacheSetInt(input.Key, input.IntVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_int":
		resp, err := host.CacheGetInt(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{IntVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_float":
		_, err := host.CacheSetFloat(input.Key, input.FloatVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_float":
		resp, err := host.CacheGetFloat(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{FloatVal: resp.Value, Exists: resp.Exists})
		return 0

	case "set_bytes":
		_, err := host.CacheSetBytes(input.Key, input.BytesVal, input.TTLSeconds)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{})
		return 0

	case "get_bytes":
		resp, err := host.CacheGetBytes(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{BytesVal: resp.Value, Exists: resp.Exists})
		return 0

	case "has":
		resp, err := host.CacheHas(input.Key)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestCacheOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestCacheOutput{Exists: resp.Exists})
		return 0

	case "remove":
		_, err := host.CacheRemove(input.Key)
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
