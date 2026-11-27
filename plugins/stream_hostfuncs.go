package plugins

import (
	"context"
	"errors"
	"io"
	"math"

	extism "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero/api"
)

// kernelLength is the Extism kernel's length export, looked up once per call: CurrentPlugin.Length looks it
// up on every use, and wazero builds a new function each time.
type kernelLength struct {
	fn    api.Function
	stack [1]uint64
}

// extismMemory adapts the calling plugin's Extism memory to guestMemory.
type extismMemory struct {
	p      *extism.CurrentPlugin
	length *kernelLength
}

func (m extismMemory) blockLength(ctx context.Context, offset uint64) uint64 {
	k := m.length
	if k.fn == nil {
		if kernel, ok := ctx.Value(extism.PluginCtxKey("extism")).(api.Module); ok {
			k.fn = kernel.ExportedFunction("length")
		}
	}
	if k.fn == nil {
		n, err := m.p.LengthWithContext(ctx, offset)
		if err != nil {
			return 0
		}
		return n
	}
	k.stack[0] = offset
	if err := k.fn.CallWithStack(ctx, k.stack[:]); err != nil {
		return 0
	}
	return k.stack[0]
}

func (m extismMemory) view(offset uint64, n uint32) ([]byte, bool) {
	if offset > math.MaxUint32 {
		return nil, false
	}
	return m.p.Memory().Read(uint32(offset), n)
}

func (m extismMemory) writeBlock(data []byte) (uint64, error) {
	return m.p.WriteBytes(data)
}

// streamHostFunctions are registered for every plugin: a handle can only come from a service the plugin already
// has permission for.
func streamHostFunctions() []extism.HostFunction {
	i64, ptr := extism.ValueTypeI64, extism.ValueTypePTR
	return []extism.HostFunction{
		streamHostFunction("stream_read", []extism.ValueType{i64, ptr, i64},
			func(ctx context.Context, s *callScope, mem guestMemory, stack []uint64) uint64 {
				return uint64(s.hostRead(ctx, mem, int64(stack[0]), stack[1], int64(stack[2])))
			}),
		streamHostFunction("stream_write", []extism.ValueType{i64, ptr, i64},
			func(ctx context.Context, s *callScope, mem guestMemory, stack []uint64) uint64 {
				return uint64(s.hostWrite(ctx, mem, int64(stack[0]), stack[1], int64(stack[2])))
			}),
		streamHostFunction("stream_copy", []extism.ValueType{i64, i64},
			func(_ context.Context, s *callScope, _ guestMemory, stack []uint64) uint64 {
				return uint64(s.hostCopy(int64(stack[0]), int64(stack[1])))
			}),
		streamHostFunction("stream_close", []extism.ValueType{i64},
			func(_ context.Context, s *callScope, _ guestMemory, stack []uint64) uint64 {
				return uint64(s.hostClose(int64(stack[0])))
			}),
		streamHostFunction("stream_error", nil,
			func(_ context.Context, s *callScope, mem guestMemory, _ []uint64) uint64 {
				return uint64(s.hostError(mem))
			}),
	}
}

// streamHostFunction makes every stream function return -1 outside a plugin call, stream_error included.
func streamHostFunction(name string, params []extism.ValueType, op func(context.Context, *callScope, guestMemory, []uint64) uint64) extism.HostFunction {
	return extism.NewHostFunctionWithStack(name,
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			if s := callScopeFrom(ctx); s != nil {
				stack[0] = op(ctx, s, extismMemory{p: p, length: &s.length}, stack)
			} else {
				stack[0] = math.MaxUint64
			}
		},
		params, []extism.ValueType{extism.ValueTypeI64})
}

const maxStreamChunk = 1 << 20

var errBadBuffer = errors.New("invalid stream buffer")

// guestMemory is the part of Extism memory the stream host functions use, so they can be tested without WASM.
type guestMemory interface {
	blockLength(ctx context.Context, offset uint64) uint64
	view(offset uint64, n uint32) ([]byte, bool)
	writeBlock(data []byte) (uint64, error)
}

func handleFromWire(v int64) (uint32, bool) {
	if v < 1 || v > math.MaxInt32 {
		return 0, false
	}
	return uint32(v), true
}

func bufferView(ctx context.Context, mem guestMemory, buf uint64, n int64) ([]byte, error) {
	if n < 1 || n > math.MaxUint32 {
		return nil, errBadBuffer
	}
	if uint64(n) > mem.blockLength(ctx, buf) {
		return nil, errBadBuffer
	}
	view, ok := mem.view(buf, uint32(n))
	if !ok {
		return nil, errBadBuffer
	}
	return view, nil
}

func (s *callScope) streamFor(handle int64, need streamRights) (*streamBackend, error) {
	h, ok := handleFromWire(handle)
	if !ok {
		return nil, errUnknownStream
	}
	return s.lookup(h, need)
}

func (s *callScope) fail(err error) int64 {
	s.lastErr = err
	return -1
}

func (s *callScope) hostRead(ctx context.Context, mem guestMemory, handle int64, buf uint64, n int64) int64 {
	b, err := s.streamFor(handle, streamRead)
	if err != nil {
		return s.fail(err)
	}
	view, err := bufferView(ctx, mem, buf, n)
	if err != nil {
		return s.fail(err)
	}
	if len(view) > maxStreamChunk {
		view = view[:maxStreamChunk]
	}
	got, err := b.Read(view)
	switch {
	case got > 0:
		s.lastErr = nil
		return int64(got)
	case errors.Is(err, io.EOF):
		s.lastErr = nil
		return 0
	default:
		return s.fail(err)
	}
}

func (s *callScope) hostWrite(ctx context.Context, mem guestMemory, handle int64, buf uint64, n int64) int64 {
	b, err := s.streamFor(handle, streamWrite)
	if err != nil {
		return s.fail(err)
	}
	view, err := bufferView(ctx, mem, buf, n)
	if err != nil {
		return s.fail(err)
	}
	written, err := b.Write(view)
	if written > 0 {
		s.lastErr = err
		return int64(written)
	}
	return s.fail(err)
}

func (s *callScope) hostCopy(dstHandle, srcHandle int64) int64 {
	dst, err := s.streamFor(dstHandle, streamWrite)
	if err != nil {
		s.lastErr = err
		return 0
	}
	src, err := s.streamFor(srcHandle, streamRead)
	if err != nil {
		s.lastErr = err
		return 0
	}
	n, err := io.Copy(dst, src)
	s.lastErr = err
	return n
}

func (s *callScope) hostClose(handle int64) int64 {
	h, ok := handleFromWire(handle)
	if !ok {
		return s.fail(errUnknownStream)
	}
	b, err := s.detach(h)
	if err != nil {
		return s.fail(err)
	}
	if err := b.Close(); err != nil {
		return s.fail(err)
	}
	s.lastErr = nil
	return 0
}

// hostError returns -1 when the message cannot be allocated (Extism signals that with offset 0, not an error),
// so the plugin never mistakes a failure for success.
func (s *callScope) hostError(mem guestMemory) int64 {
	if s.lastErr == nil {
		return 0
	}
	msg := s.lastErr.Error()
	if msg == "" {
		msg = "stream operation failed"
	}
	off, err := mem.writeBlock([]byte(msg))
	if err != nil || off == 0 || off > math.MaxInt64 {
		return -1
	}
	return int64(off)
}
