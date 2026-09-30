// Test plugin for generated host functions that take and return streams.
package main

import (
	"io"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"

	"test-stream-service/host"
)

type input struct {
	Text string `json:"text"`
}

type output struct {
	Data  string `json:"data,omitempty"`
	Size  int64  `json:"size,omitempty"`
	Error string `json:"error,omitempty"`
}

func run(fn func(in input) output) int32 {
	var in input
	if err := pdk.InputJSON(&in); err != nil {
		pdk.SetError(err)
		return 1
	}
	if err := pdk.OutputJSON(fn(in)); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

func fail(err error) output { return output{Error: err.Error()} }

//go:wasmexport open_read
func openRead() int32 {
	return run(func(in input) output {
		rc, err := host.StreamTestOpen(in.Text)
		if err != nil {
			return fail(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return fail(err)
		}
		return output{Data: string(data)}
	})
}

//go:wasmexport create_write
func createWrite() int32 {
	return run(func(in input) output {
		wc, err := host.StreamTestCreate("out")
		if err != nil {
			return fail(err)
		}
		if _, err := wc.Write([]byte(in.Text)); err != nil {
			return fail(err)
		}
		if err := wc.Close(); err != nil {
			return fail(err)
		}
		return output{}
	})
}

// drain_own passes a host stream to Drain, then tries to read it again.
//
//go:wasmexport drain_own
func drainOwn() int32 {
	return run(func(in input) output {
		rc, err := host.StreamTestOpen(in.Text)
		if err != nil {
			return fail(err)
		}
		data, err := host.StreamTestDrain(rc)
		if err != nil {
			return fail(err)
		}
		_, readErr := rc.Read(make([]byte, 1))
		return output{Data: data, Error: readErr.Error()}
	})
}

//go:wasmexport drain_plugin_reader
func drainPluginReader() int32 {
	return run(func(in input) output {
		_, err := host.StreamTestDrain(io.NopCloser(strings.NewReader(in.Text)))
		return fail(err)
	})
}

//go:wasmexport drain_nil
func drainNil() int32 {
	return run(func(in input) output {
		data, err := host.StreamTestDrain(nil)
		if err != nil {
			return fail(err)
		}
		return output{Data: data}
	})
}

//go:wasmexport drain_writer
func drainWriter() int32 {
	return run(func(in input) output {
		wc, err := host.StreamTestCreate("out")
		if err != nil {
			return fail(err)
		}
		_, err = host.StreamTestDrain(wc.(io.ReadCloser))
		return fail(err)
	})
}

//go:wasmexport wrap
func wrap() int32 {
	return run(func(in input) output {
		body, err := host.StreamTestOpen(in.Text)
		if err != nil {
			return fail(err)
		}
		resp, err := host.StreamTestWrap(host.WrapRequest{Prefix: "p:", Body: body})
		if err != nil {
			return fail(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fail(err)
		}
		return output{Data: string(data), Size: resp.Size}
	})
}

//go:wasmimport extism:host/user streamtest_drain
func rawDrain(uint64) uint64

// drain_null_raw sends a JSON null handle, as a dynamic-language plugin might.
//
//go:wasmexport drain_null_raw
func drainNullRaw() int32 {
	return run(func(in input) output {
		req := pdk.AllocateBytes([]byte(`{"in":null}`))
		defer req.Free()
		mem := pdk.FindMemory(rawDrain(req.Offset()))
		data := mem.ReadBytes()
		mem.Free()
		return output{Data: string(data)}
	})
}

// drain_blocked hands the service a stream that never delivers data.
//
//go:wasmexport drain_blocked
func drainBlocked() int32 {
	return run(func(in input) output {
		rc, err := host.StreamTestBlock()
		if err != nil {
			return fail(err)
		}
		_, err = host.StreamTestDrain(rc)
		return fail(err)
	})
}

// drain_loop reads from and then hands over many streams, so a buffer kept per handed-over stream runs out
// of memory.
//
//go:wasmexport drain_loop
func drainLoop() int32 {
	return run(func(in input) output {
		for i := range 2000 {
			rc, err := host.StreamTestOpen(in.Text)
			if err != nil {
				return fail(err)
			}
			if _, err := rc.Read(make([]byte, 1)); err != nil {
				return fail(err)
			}
			if _, err := host.StreamTestDrain(rc); err != nil {
				return output{Error: err.Error(), Size: int64(i)}
			}
		}
		return output{Size: 2000}
	})
}

func main() {}
