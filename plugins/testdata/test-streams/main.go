// Test plugin for stream host functions, the generated StreamTest host functions, and call time limits.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/stream"
)

type input struct {
	Handle uint32 `json:"handle"`
	Dst    uint32 `json:"dst"`
	Size   int64  `json:"size"`
	Chunk  int    `json:"chunk"`
	Text   string `json:"text"`
}

type output struct {
	Count int64  `json:"count"`
	SHA   string `json:"sha,omitempty"`
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

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// read_all reads the stream to the end and reports its size, SHA-256, and first KiB.
//
//go:wasmexport read_all
func readAll() int32 {
	return run(func(in input) output {
		s := stream.New(in.Handle)
		h := sha256.New()
		buf := make([]byte, max(in.Chunk, 1))
		var out output
		var prefix []byte
		for {
			n, err := s.Read(buf)
			h.Write(buf[:n])
			out.Count += int64(n)
			if len(prefix) < 1024 {
				prefix = append(prefix, buf[:min(n, 1024-len(prefix))]...)
			}
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				out.Error = err.Error()
				break
			}
		}
		out.SHA = hex.EncodeToString(h.Sum(nil))
		out.Data = string(prefix)
		return out
	})
}

// write_pattern writes Size bytes of (offset % 251) in Chunk-sized writes, then closes the stream.
//
//go:wasmexport write_pattern
func writePattern() int32 {
	return run(func(in input) output {
		s := stream.New(in.Handle)
		chunk := make([]byte, max(in.Chunk, 1))
		var out output
		for out.Count < in.Size {
			n := int(min(int64(len(chunk)), in.Size-out.Count))
			for i := range n {
				chunk[i] = byte((out.Count + int64(i)) % 251)
			}
			w, err := s.Write(chunk[:n])
			out.Count += int64(w)
			if err != nil {
				out.Error = err.Error()
				return out
			}
		}
		out.Error = errString(s.Close())
		return out
	})
}

//go:wasmexport copy
func copyStreams() int32 {
	return run(func(in input) output {
		n, err := stream.Copy(stream.New(in.Dst), stream.New(in.Handle))
		return output{Count: n, Error: errString(err)}
	})
}

//go:wasmexport close_twice
func closeTwice() int32 {
	return run(func(in input) output {
		s := stream.New(in.Handle)
		first := s.Close()
		second := s.Close()
		return output{Error: errString(first) + "|" + errString(second)}
	})
}

// leak reads one byte and returns without closing the stream.
//
//go:wasmexport leak
func leak() int32 {
	return run(func(in input) output {
		n, err := stream.New(in.Handle).Read(make([]byte, 1))
		return output{Count: int64(n), Error: errString(err)}
	})
}

// write_when_out_of_memory fills Extism's block memory before it writes, so the stream buffer cannot be allocated.
//
//go:wasmexport write_when_out_of_memory
func writeWhenOutOfMemory() int32 {
	return run(func(in input) output {
		var blocks []pdk.Memory
		for range 1 << 16 {
			m := pdk.Allocate(64 << 10)
			if m.Offset() == 0 {
				break
			}
			blocks = append(blocks, m)
		}
		_, err := stream.New(in.Handle).Write([]byte("x"))
		for _, m := range blocks {
			m.Free()
		}
		return output{Error: errString(err)}
	})
}

//go:wasmexport spin
func spin() int32 {
	for {
	}
}

//go:wasmexport noop
func noop() int32 {
	return 0
}

func main() {}
