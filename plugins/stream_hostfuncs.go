package plugins

import (
	"context"
	"math"

	extism "github.com/extism/go-sdk"
)

// extismMemory adapts the calling plugin's Extism memory to guestMemory.
type extismMemory struct{ p *extism.CurrentPlugin }

func (m extismMemory) blockLength(offset uint64) uint64 {
	n, err := m.p.Length(offset)
	if err != nil {
		return 0
	}
	return n
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
			func(s *callScope, mem guestMemory, stack []uint64) uint64 {
				return uint64(s.hostRead(mem, int64(stack[0]), stack[1], int64(stack[2])))
			}),
		streamHostFunction("stream_write", []extism.ValueType{i64, ptr, i64},
			func(s *callScope, mem guestMemory, stack []uint64) uint64 {
				return uint64(s.hostWrite(mem, int64(stack[0]), stack[1], int64(stack[2])))
			}),
		streamHostFunction("stream_copy", []extism.ValueType{i64, i64},
			func(s *callScope, _ guestMemory, stack []uint64) uint64 {
				return uint64(s.hostCopy(int64(stack[0]), int64(stack[1])))
			}),
		streamHostFunction("stream_close", []extism.ValueType{i64},
			func(s *callScope, _ guestMemory, stack []uint64) uint64 {
				return uint64(s.hostClose(int64(stack[0])))
			}),
		streamHostFunction("stream_error", nil,
			func(s *callScope, mem guestMemory, _ []uint64) uint64 {
				return uint64(s.hostError(mem))
			}),
	}
}

// Outside a plugin call there is no scope: every function returns -1 and stream_error has no message.
func streamHostFunction(name string, params []extism.ValueType, op func(*callScope, guestMemory, []uint64) uint64) extism.HostFunction {
	return extism.NewHostFunctionWithStack(name,
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			s := callScopeFrom(ctx)
			switch {
			case s != nil:
				stack[0] = op(s, extismMemory{p}, stack)
			case name == "stream_error":
				stack[0] = 0
			default:
				stack[0] = math.MaxUint64
			}
		},
		params, []extism.ValueType{extism.ValueTypeI64})
}
