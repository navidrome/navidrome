package pglite

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const errnoNotsup = 58

// switchableStdin is the guest's stdin. PGlite runs initdb by dup2()-ing a script file onto
// fd 0, which wazero refuses for preopens, so we emulate it by swapping the reader instead.
type switchableStdin struct {
	mu sync.Mutex
	r  io.ReadCloser
}

func (s *switchableStdin) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.r == nil {
		return 0, io.EOF
	}
	return s.r.Read(p)
}

func (s *switchableStdin) set(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.r != nil {
		_ = s.r.Close()
	}
	s.r = f
	return nil
}

func (s *switchableStdin) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.r != nil {
		err := s.r.Close()
		s.r = nil
		return err
	}
	return nil
}

// fdTracker listens to path_open so fd_renumber can resolve a guest fd back to a host path.
type fdTracker struct {
	mounts  map[int32]string // preopen fd -> host dir
	opened  map[int32]string // guest fd -> host path
	pending string
	outPtr  uint32
	pathBad bool
}

func newFDTracker(mounts ...string) *fdTracker {
	t := &fdTracker{mounts: map[int32]string{}, opened: map[int32]string{}}
	for i, m := range mounts {
		t.mounts[int32(3+i)] = m
	}
	return t
}

func (t *fdTracker) NewFunctionListener(def api.FunctionDefinition) experimental.FunctionListener {
	if def.ModuleName() == wasi_snapshot_preview1.ModuleName && def.Name() == "path_open" {
		return t
	}
	return nil
}

func (t *fdTracker) Before(_ context.Context, mod api.Module, _ api.FunctionDefinition, params []uint64, _ experimental.StackIterator) {
	dir, ok := t.mounts[int32(params[0])]
	path, okMem := mod.Memory().Read(uint32(params[2]), uint32(params[3]))
	t.pathBad = !ok || !okMem
	if !t.pathBad {
		t.pending = filepath.Join(dir, string(path))
	}
	t.outPtr = uint32(params[8])
}

func (t *fdTracker) After(_ context.Context, mod api.Module, _ api.FunctionDefinition, results []uint64) {
	if t.pathBad || results[0] != 0 {
		return
	}
	if fd, ok := mod.Memory().ReadUint32Le(t.outPtr); ok {
		t.opened[int32(fd)] = t.pending
	}
}

func (t *fdTracker) Abort(context.Context, api.Module, api.FunctionDefinition, error) {}

// instantiateWASI installs wasi_snapshot_preview1 with fd_renumber(fd, 0) emulated via stdin.
func instantiateWASI(ctx context.Context, r wazero.Runtime, tracker *fdTracker, stdin *switchableStdin) error {
	b := r.NewHostModuleBuilder(wasi_snapshot_preview1.ModuleName)
	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(b)
	b.NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module, from, to int32) int32 {
			if to != 0 {
				return errnoNotsup
			}
			path, ok := tracker.opened[from]
			if !ok {
				return errnoNotsup
			}
			if err := stdin.set(path); err != nil {
				return errnoNotsup
			}
			return 0
		}).
		Export("fd_renumber")
	_, err := b.Instantiate(ctx)
	return err
}
