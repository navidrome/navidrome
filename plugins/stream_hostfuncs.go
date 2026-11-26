package plugins

import (
	"context"
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
		n, err := m.p.Length(offset)
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
