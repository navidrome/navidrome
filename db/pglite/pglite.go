// Package pglite runs the PGlite WASI build of PostgreSQL inside the process using wazero.
// Spike: ported from github.com/elliots/go-pglite (wasmtime) to wazero.
package pglite

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/sys"
)

type Config struct {
	DataDir  string
	Tarball  string // path to pglite-wasi-17.tar.gz
	Database string
	User     string
	Stderr   io.Writer
	// ServerSettings are appended to postgresql.conf after initdb (e.g. fsync=off).
	ServerSettings map[string]string
	// ExtraArgs are appended to the postgres argv (e.g. "-c", "fsync=off").
	ExtraArgs []string
	// WasmOverride, if set, is used instead of the pglite.wasi extracted from the tarball.
	WasmOverride string
	// SocketDir is where the unix socket is created. Defaults to a temp dir removed on Close.
	SocketDir string
}

type PGlite struct {
	wasmMu sync.Mutex
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc

	runtime wazero.Runtime
	mod     api.Module
	stdin   *switchableStdin
	trace   bool
	v2      bool

	connections   atomic.Int64
	sessionMu     sync.Mutex    // one backend means one session: serialize exchanges, and whole transactions
	tFiles, tWasm time.Duration // trace-mode timing accumulators
	ticks         int
	paramStatus   []byte // ParameterStatus messages from the first handshake, replayed on later ones

	dataDir       string
	ioBase        string
	tempSocketDir bool
	socketDir     string
	socketPath    string
	listener      net.Listener
	wg            sync.WaitGroup

	fnInteractiveOne   api.Function
	fnInteractiveWrite api.Function
	fnInteractiveRead  api.Function
	fnGetChannel       api.Function

	// Shared-memory ("CMA") transport: wire bytes go straight into wasm memory instead of
	// through the .in/.out files. Falls back to files when the module does not offer it.
	cmaOK          bool
	cmaAddr        uint32
	pendingWireLen uint32
	fnUseWire      api.Function
	fnClearError   api.Function
}

func New(ctx context.Context, cfg Config) (*PGlite, error) {
	if cfg.Database == "" {
		cfg.Database = "postgres"
	}
	if cfg.User == "" {
		cfg.User = "postgres"
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	ctx, cancel := context.WithCancel(ctx)
	pg := &PGlite{cfg: cfg, ctx: ctx, cancel: cancel, dataDir: cfg.DataDir, trace: os.Getenv("ND_PGLITE_TRACE") != ""}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		cancel()
		return nil, err
	}
	if WASIBinary == nil {
		b, err := os.ReadFile(cfg.Tarball)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("reading tarball: %w", err)
		}
		WASIBinary = b
	}
	wasmBinary, err := setupEnvironment(pg.dataDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("setup environment: %w", err)
	}
	if cfg.WasmOverride != "" {
		if wasmBinary, err = os.ReadFile(cfg.WasmOverride); err != nil {
			cancel()
			return nil, fmt.Errorf("reading wasm override: %w", err)
		}
	}

	// v2 layout (pglite4j build): share files embedded via wasi-vfs, pre-initialized
	// cluster shipped as pgdata/, initdb+backend already run by wizer.
	pg.v2 = fileExists(filepath.Join(pg.dataDir, "pgdata", "PG_VERSION"))
	pgdataDir := filepath.Join(pg.dataDir, "pglite", "base")
	if pg.v2 {
		pgdataDir = filepath.Join(pg.dataDir, "pgdata")
	}
	devDir := filepath.Join(pg.dataDir, "dev")
	if err := os.MkdirAll(pgdataDir, 0o755); err != nil {
		cancel()
		return nil, err
	}

	start := time.Now()
	pg.stdin = &switchableStdin{}
	tracker := newFDTracker(pg.dataDir, pgdataDir, devDir)
	_ = tracker
	ctx = experimental.WithFunctionListenerFactory(ctx, tracker)
	pg.runtime = wazero.NewRuntime(ctx)
	if err := instantiateWASI(ctx, pg.runtime, tracker, pg.stdin); err != nil {
		pg.Close()
		return nil, fmt.Errorf("instantiating WASI: %w", err)
	}
	compiled, err := pg.runtime.CompileModule(ctx, wasmBinary)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("compiling pglite.wasi: %w", err)
	}
	compileTime := time.Since(start)

	modCfg := wazero.NewModuleConfig().
		WithArgs(append([]string{"/tmp/pglite/bin/postgres", "--single"}, append(cfg.ExtraArgs, cfg.Database)...)...).
		WithEnv("ENVIRONMENT", "wasm32_wasi_preview1").
		WithEnv("PREFIX", "/tmp/pglite").
		WithEnv("PGDATA", pg.guestPGData()).
		WithEnv("PGSYSCONFDIR", "/tmp/pglite").
		WithEnv("PGUSER", cfg.User).
		WithEnv("PGDATABASE", cfg.Database).
		WithEnv("MODE", "REACT").
		WithEnv("REPL", "N").
		WithEnv("TZ", "UTC").
		WithEnv("PGTZ", "UTC").
		WithEnv("PATH", "/tmp/pglite/bin").
		WithFSConfig(wazero.NewFSConfig().
			WithDirMount(pg.dataDir, "/tmp").
			WithDirMount(pgdataDir, pg.guestPGData()). // preopen order must match the wizer snapshot
			WithDirMount(devDir, "/dev")).
		WithStdin(pg.stdin).
		WithStdout(io.Discard).
		WithStderr(cfg.Stderr).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithRandSource(rand.Reader).
		WithStartFunctions() // call _start ourselves so an exit does not abort instantiation

	pg.mod, err = pg.runtime.InstantiateModule(ctx, compiled, modCfg)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("instantiating pglite.wasi: %w", err)
	}

	startFns := []string{"_start", "pgl_initdb", "pgl_backend"}
	if pg.v2 {
		startFns = nil // wizer already ran them at build time
	}
	for _, name := range startFns {
		fn := pg.mod.ExportedFunction(name)
		if fn == nil {
			continue
		}
		if _, err := fn.Call(ctx); err != nil {
			var exitErr *sys.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 0 {
				continue
			}
			pg.Close()
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if name == "pgl_initdb" && len(cfg.ServerSettings) > 0 {
			if err := appendSettings(filepath.Join(pgdataDir, "postgresql.conf"), cfg.ServerSettings); err != nil {
				pg.Close()
				return nil, err
			}
		}
	}
	if pg.v2 {
		if fn := pg.mod.ExportedFunction("interactive_write"); fn != nil {
			if _, err := fn.Call(ctx, api.EncodeI32(0)); err != nil {
				pg.Close()
				return nil, fmt.Errorf("interactive_write(0): %w", err)
			}
		}
	}
	fmt.Fprintf(cfg.Stderr, "# pglite: wazero compile=%s init=%s v2=%v\n", compileTime, time.Since(start), pg.v2)

	pg.fnInteractiveOne = pg.mod.ExportedFunction("interactive_one")
	pg.fnInteractiveWrite = pg.mod.ExportedFunction("interactive_write")
	pg.fnInteractiveRead = pg.mod.ExportedFunction("interactive_read")
	pg.fnGetChannel = pg.mod.ExportedFunction("get_channel")
	pg.fnUseWire = pg.mod.ExportedFunction("use_wire")
	pg.fnClearError = pg.mod.ExportedFunction("clear_error")
	pg.probeCMA(ctx)
	fmt.Fprintf(cfg.Stderr, "# pglite: transport=%s\n", map[bool]string{true: "shared-memory", false: "files"}[pg.cmaOK])
	if pg.fnInteractiveOne == nil {
		pg.Close()
		return nil, errors.New("module missing 'interactive_one' export")
	}

	if err := pg.startBridge(); err != nil {
		pg.Close()
		return nil, fmt.Errorf("starting socket bridge: %w", err)
	}
	return pg, nil
}

func (pg *PGlite) guestPGData() string {
	if pg.v2 {
		return "/pgdata"
	}
	return "/tmp/pglite/base"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func appendSettings(confPath string, settings map[string]string) error {
	f, err := os.OpenFile(confPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening postgresql.conf: %w", err)
	}
	defer f.Close()
	for k, v := range settings {
		if _, err := fmt.Fprintf(f, "\n%s = %s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

// DSN returns a pgx connection string for the embedded instance. Simple protocol only:
// the WASI build does not cope with extended-protocol portals.
func (pg *PGlite) DSN() string {
	return fmt.Sprintf("host=%s port=5432 dbname=%s user=%s sslmode=disable default_query_exec_mode=simple_protocol",
		pg.socketDir, pg.cfg.Database, pg.cfg.User)
}

// OpenDB returns a pool that reaches the backend over an in-process pipe, skipping the unix
// socket. The socket stays up for external tools.
func (pg *PGlite) OpenDB() (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(pg.DSN())
	if err != nil {
		return nil, err
	}
	cfg.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		pg.wg.Add(1)
		go func() {
			defer pg.wg.Done()
			pg.handleConn(server, pg.ioBase)
		}()
		return client, nil
	}
	return stdlib.OpenDB(*cfg), nil
}

func (pg *PGlite) Close() error {
	pg.cancel()
	if pg.listener != nil {
		_ = pg.listener.Close()
	}
	pg.wg.Wait()
	if pg.socketDir != "" {
		if pg.tempSocketDir {
			_ = os.RemoveAll(pg.socketDir)
		} else {
			_ = os.Remove(pg.socketPath)
		}
	}
	if pg.stdin != nil {
		_ = pg.stdin.Close()
	}
	if pg.runtime != nil {
		return pg.runtime.Close(context.Background())
	}
	return nil
}

func (pg *PGlite) startBridge() error {
	sockDir := pg.cfg.SocketDir
	if sockDir == "" {
		var err error
		if sockDir, err = os.MkdirTemp("", "pglite-sock-*"); err != nil {
			return err
		}
		pg.tempSocketDir = true
	} else {
		if err := os.MkdirAll(sockDir, 0o700); err != nil {
			return err
		}
		// pgx only treats a host as a unix socket when the path is absolute.
		abs, err := filepath.Abs(sockDir)
		if err != nil {
			return err
		}
		sockDir = abs
		_ = os.Remove(filepath.Join(sockDir, ".s.PGSQL.5432")) // a stale socket from a crash
	}
	pg.socketDir = sockDir
	pg.socketPath = filepath.Join(sockDir, ".s.PGSQL.5432")
	ln, err := net.Listen("unix", pg.socketPath)
	if err != nil {
		return err
	}
	pg.listener = ln

	ioBase := filepath.Join(pg.dataDir, "pglite", "base", ".s.PGSQL.5432")
	if pg.v2 {
		ioBase = filepath.Join(pg.dataDir, "pgdata", ".s.PGSQL.5432")
	}
	pg.ioBase = ioBase
	for _, suffix := range []string{".in", ".out", ".lock.in", ".lock.out"} {
		_ = os.Remove(ioBase + suffix)
	}

	pg.wg.Add(1)
	go func() {
		defer pg.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			pg.wg.Add(1)
			go func() {
				defer pg.wg.Done()
				pg.handleConn(conn, ioBase)
			}()
		}
	}()
	return nil
}

// Connections reports how many client connections the bridge has accepted.
func (pg *PGlite) Connections() int64 { return pg.connections.Load() }

func (pg *PGlite) handleConn(conn net.Conn, ioBase string) {
	pg.connections.Add(1)
	inTx, holding := false, false
	defer func() {
		if holding {
			if inTx {
				pg.rollback(ioBase)
			}
			pg.sessionMu.Unlock()
		}
	}()
	handshakeDone := false
	startupDone := false
	var pending []byte
	if pg.trace {
		fmt.Fprintln(pg.cfg.Stderr, "# bridge: client connected")
		defer fmt.Fprintln(pg.cfg.Stderr, "# bridge: client disconnected")
	}
	defer conn.Close()
	outFile := ioBase + ".out"
	buf := make([]byte, 65536)

	for {
		select {
		case <-pg.ctx.Done():
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(16 * time.Millisecond))
		n, readErr := conn.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			n = completeMessages(pending, &startupDone)
		}
		if n > 0 {
			packet := pending[:n:n]
			pending = append([]byte(nil), pending[n:]...)
			if pg.trace {
				fmt.Fprintf(pg.cfg.Stderr, "# bridge C> %s (%d bytes)\n", wireTags(packet), n)
			}
			// Take the session before touching .in: it is shared by every client.
			if !holding {
				pg.sessionMu.Lock()
				holding = true
			}
			pg.wasmMu.Lock()
			t0 := time.Now()
			replies, trapErr := pg.forwardWire(packet, outFile)
			if pg.trace {
				fmt.Fprintf(pg.cfg.Stderr, "# bridge timing: total=%s files=%s wasm=%s ticks=%d\n",
					time.Since(t0).Round(time.Microsecond), pg.tFiles.Round(time.Microsecond), pg.tWasm.Round(time.Microsecond), pg.ticks)
				pg.tFiles, pg.tWasm, pg.ticks = 0, 0, 0
			}
			pg.wasmMu.Unlock()
			if !handshakeDone {
				replies, handshakeDone = pg.fixHandshake(replies)
			} else {
				replies = ensureReadyForQuery(replies, trapErr)
			}
			if trapErr != nil && pg.trace {
				fmt.Fprintf(pg.cfg.Stderr, "# bridge: trap recovered: %v\n", strings.SplitN(trapErr.Error(), "\n", 2)[0])
			}
			// Keep the session across a multi-step handshake and across a transaction: both are
			// stateful in the one backend, so another client must not interleave into them.
			status := lastReadyStatus(replies)
			inTx = status == 'T' || status == 'E'
			if handshakeDone && !inTx {
				pg.sessionMu.Unlock()
				holding = false
			}
			// A PG ERROR surfaces as a trap here; the reply is already complete, so keep the client.
			if !pg.sendReplies(conn, replies) {
				return
			}
		}
		if readErr != nil {
			var netErr net.Error
			if errors.As(readErr, &netErr) && netErr.Timeout() {
				continue
			}
			return
		}
	}
}

func (pg *PGlite) forwardWire(packet []byte, outFile string) ([][]byte, error) {
	const maxTicks = 256
	ctx := pg.ctx
	if len(packet) > 0 {
		if err := pg.send(packet, strings.TrimSuffix(outFile, ".out")); err != nil {
			return nil, err
		}
	}
	if pg.fnUseWire != nil {
		_, _ = pg.fnUseWire.Call(ctx, api.EncodeI32(1))
	}
	var replies [][]byte
	for range maxTicks {
		producedBefore := pg.collectReply(outFile, &replies)
		t0 := time.Now()
		_, err := pg.fnInteractiveOne.Call(ctx)
		pg.tWasm += time.Since(t0)
		pg.ticks++
		if pg.trace {
			fmt.Fprintf(pg.cfg.Stderr, "# bridge tick %d: %s\n", pg.ticks, time.Since(t0).Round(time.Microsecond))
		}
		if err != nil && pg.v2 {
			// v2 build: pgl_on_error sets a flag and traps; clear_error does the full cleanup.
			pg.collectReply(outFile, &replies)
			if fn := pg.mod.ExportedFunction("pgl_check_error"); fn != nil {
				res, cerr := fn.Call(ctx)
				if pg.trace {
					fmt.Fprintf(pg.cfg.Stderr, "# bridge v2 trap: pgl_check_error=%v err=%v\n", res, cerr)
				}
				if cerr == nil && len(res) > 0 && api.DecodeI32(res[0]) != 0 {
					if pg.fnClearError != nil {
						_, _ = pg.fnClearError.Call(ctx)
					}
					if pg.fnInteractiveWrite != nil {
						_, _ = pg.fnInteractiveWrite.Call(ctx, api.EncodeI32(-1))
					}
					_, _ = pg.fnInteractiveOne.Call(ctx)
					pg.collectReply(outFile, &replies)
				}
			}
			return replies, err
		}
		if err != nil {
			pg.collectReply(outFile, &replies)
			// Traps bypass PG_CATCH; flag shmem_exit_inprogress so clear_error drops active portals.
			const shmemExitAddr = 4895117
			pg.mod.Memory().WriteByte(shmemExitAddr, 1)
			if pg.fnClearError != nil {
				_, _ = pg.fnClearError.Call(ctx)
			}
			pg.mod.Memory().WriteByte(shmemExitAddr, 0)
			_ = os.Remove(strings.TrimSuffix(outFile, ".out") + ".in")
			if pg.fnInteractiveWrite != nil {
				_, _ = pg.fnInteractiveWrite.Call(ctx, api.EncodeI32(-1))
			}
			if pg.fnUseWire != nil {
				_, _ = pg.fnUseWire.Call(ctx, api.EncodeI32(1))
			}
			_, _ = pg.fnInteractiveOne.Call(ctx)
			pg.collectReply(outFile, &replies)
			return replies, err
		}
		producedAfter := pg.collectReply(outFile, &replies)
		if !producedBefore && !producedAfter {
			break
		}
		if endsWithReadyForQuery(replies) {
			break // a complete response; skip the empty probe tick
		}
	}
	return replies, nil
}

func (pg *PGlite) collectReply(outFile string, replies *[][]byte) bool {
	t0 := time.Now()
	defer func() { pg.tFiles += time.Since(t0) }()
	if pg.cmaOK {
		// A negative channel means the C side put this reply in a file after all.
		if res, err := pg.fnGetChannel.Call(pg.ctx); err == nil && len(res) > 0 && api.DecodeI32(res[0]) >= 0 {
			return pg.collectFromMemory(replies)
		}
	}
	data, err := os.ReadFile(outFile)
	if err != nil || len(data) == 0 {
		return false
	}
	_ = os.Remove(outFile)
	*replies = append(*replies, data)
	return true
}

// collectFromMemory reads one reply out of the shared wire buffer. Must be called with wasmMu held.
func (pg *PGlite) collectFromMemory(replies *[][]byte) bool {
	res, err := pg.fnInteractiveRead.Call(pg.ctx)
	if err != nil || len(res) == 0 {
		return false
	}
	n := api.DecodeI32(res[0])
	if n <= 0 {
		return false
	}
	data, ok := pg.mod.Memory().Read(pg.cmaAddr+pg.pendingWireLen+1, uint32(n))
	if !ok {
		return false
	}
	*replies = append(*replies, bytes.Clone(data))
	_, _ = pg.fnInteractiveWrite.Call(pg.ctx, api.EncodeI32(0))
	pg.pendingWireLen = 0
	return true
}

func (pg *PGlite) sendReplies(conn net.Conn, replies [][]byte) bool {
	for _, data := range replies {
		if len(data) == 0 {
			continue
		}
		if pg.trace {
			fmt.Fprintf(pg.cfg.Stderr, "# bridge S> %s (%d bytes)\n", wireTags(data), len(data))
			if data[0] == 'E' || data[0] == 'N' {
				fmt.Fprintf(pg.cfg.Stderr, "# bridge S> raw %q\n", data)
			}
		}
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := conn.Write(data); err != nil {
			return false
		}
	}
	return true
}

// completeMessages returns how many leading bytes of data form whole client messages. The first
// message of a connection (startup/cancel) is untagged; every later one is tag + int32 length.
func completeMessages(data []byte, startupDone *bool) int {
	n := 0
	for {
		rest := data[n:]
		if !*startupDone {
			if len(rest) < 4 {
				return n
			}
			size := int(rest[0])<<24 | int(rest[1])<<16 | int(rest[2])<<8 | int(rest[3])
			if size < 4 || size > len(rest) {
				return n
			}
			n += size
			*startupDone = true
			continue
		}
		if len(rest) < 5 {
			return n
		}
		size := int(rest[1])<<24 | int(rest[2])<<16 | int(rest[3])<<8 | int(rest[4])
		if size < 4 || size+1 > len(rest) {
			return n
		}
		n += size + 1
	}
}

func endsWithReadyForQuery(replies [][]byte) bool {
	if len(replies) == 0 {
		return false
	}
	last := replies[len(replies)-1]
	return len(last) >= 6 && last[len(last)-6] == 'Z' && last[len(last)-5] == 0 && last[len(last)-2] == 5
}

// lastReadyStatus returns the status byte of the final ReadyForQuery in replies, or 0 if there is none.
func lastReadyStatus(replies [][]byte) byte {
	data := bytes.Join(replies, nil)
	var status byte
	for rest := data; len(rest) >= 5; {
		n := int(rest[1])<<24 | int(rest[2])<<16 | int(rest[3])<<8 | int(rest[4])
		if n < 4 || n+1 > len(rest) {
			break
		}
		if rest[0] == 'Z' && n == 5 {
			status = rest[5]
		}
		rest = rest[n+1:]
	}
	return status
}

// probeCMA asks the module for its shared-memory wire buffer. A negative channel means the
// module only speaks the file transport.
func (pg *PGlite) probeCMA(ctx context.Context) {
	if os.Getenv("ND_PGLITE_NOCMA") != "" {
		return
	}
	addrFn := pg.mod.ExportedFunction("get_buffer_addr")
	if pg.fnInteractiveRead == nil || pg.fnInteractiveWrite == nil || pg.fnGetChannel == nil || addrFn == nil {
		return
	}
	if _, err := pg.fnInteractiveWrite.Call(ctx, api.EncodeI32(0)); err != nil {
		return
	}
	res, err := pg.fnGetChannel.Call(ctx)
	if err != nil || len(res) == 0 || api.DecodeI32(res[0]) < 0 {
		return
	}
	channel := api.DecodeI32(res[0])
	addr, err := addrFn.Call(ctx, api.EncodeI32(channel))
	if err != nil || len(addr) == 0 || api.DecodeI32(addr[0]) <= 0 {
		return
	}
	pg.cmaAddr = uint32(api.DecodeI32(addr[0]))
	pg.cmaOK = true
}

// send hands one client packet to the backend. Must be called with wasmMu held.
func (pg *PGlite) send(packet []byte, ioBase string) error {
	if pg.cmaOK {
		if pg.fnUseWire != nil {
			if _, err := pg.fnUseWire.Call(pg.ctx, api.EncodeI32(1)); err != nil {
				return err
			}
		}
		if !pg.mod.Memory().Write(pg.cmaAddr, packet) {
			return fmt.Errorf("wire buffer too small for %d bytes", len(packet))
		}
		if _, err := pg.fnInteractiveWrite.Call(pg.ctx, api.EncodeI32(int32(len(packet)))); err != nil {
			return err
		}
		pg.pendingWireLen = uint32(len(packet))
		return nil
	}
	if err := os.WriteFile(ioBase+".lock.in", packet, 0o644); err != nil {
		return err
	}
	return os.Rename(ioBase+".lock.in", ioBase+".in")
}

// rollback aborts a transaction left open by a client that disconnected mid-transaction, so the
// next client does not inherit it. Must be called with sessionMu held.
func (pg *PGlite) rollback(ioBase string) {
	sql := "ROLLBACK\x00"
	msg := append([]byte{'Q'}, byte((len(sql)+4)>>24), byte((len(sql)+4)>>16), byte((len(sql)+4)>>8), byte(len(sql)+4))
	msg = append(msg, sql...)
	pg.wasmMu.Lock()
	_, _ = pg.forwardWire(msg, ioBase+".out")
	pg.wasmMu.Unlock()
}

// fixHandshake caches the ParameterStatus ('S') messages of the first handshake and injects them
// into later ones: PGlite only sends them once per process, but every new pgx connection needs them.
func (pg *PGlite) fixHandshake(replies [][]byte) ([][]byte, bool) {
	data := bytes.Join(replies, nil)
	var params []byte
	ready := false
	for rest := data; len(rest) >= 5; {
		n := int(rest[1])<<24 | int(rest[2])<<16 | int(rest[3])<<8 | int(rest[4])
		if n < 4 || n+1 > len(rest) {
			break
		}
		switch rest[0] {
		case 'S':
			params = append(params, rest[:n+1]...)
		case 'Z':
			ready = true
		}
		rest = rest[n+1:]
	}
	if !ready {
		return replies, false
	}
	if len(params) > 0 {
		pg.paramStatus = params
		return replies, true
	}
	if pg.paramStatus == nil {
		return replies, true
	}
	// No 'S' messages: insert the cached ones before the first 'K' or 'Z'.
	for i := 0; i+5 <= len(data); {
		n := int(data[i+1])<<24 | int(data[i+2])<<16 | int(data[i+3])<<8 | int(data[i+4])
		if data[i] == 'K' || data[i] == 'Z' {
			patched := append(append(append([]byte{}, data[:i]...), pg.paramStatus...), data[i:]...)
			return [][]byte{patched}, true
		}
		i += n + 1
	}
	return replies, true
}

// ensureReadyForQuery appends a ReadyForQuery when PGlite ends an error reply without one,
// which would otherwise leave the client waiting forever.
func ensureReadyForQuery(replies [][]byte, trapErr error) [][]byte {
	data := bytes.Join(replies, nil)
	if len(data) == 0 && trapErr == nil {
		return replies
	}
	sawError, sawReady := false, false
	for rest := data; len(rest) >= 5; {
		n := int(rest[1])<<24 | int(rest[2])<<16 | int(rest[3])<<8 | int(rest[4])
		if n < 4 || n+1 > len(rest) {
			break
		}
		switch rest[0] {
		case 'E':
			sawError = true
		case 'Z':
			sawReady = true
		}
		rest = rest[n+1:]
	}
	if trapErr != nil && !sawError && !sawReady {
		replies = append(replies, errorResponse("XX000", "pglite trap: "+strings.SplitN(trapErr.Error(), "\n", 2)[0]))
		sawError = true
	}
	if sawError && !sawReady {
		return append(replies, []byte{'Z', 0, 0, 0, 5, 'I'})
	}
	return replies
}

func errorResponse(code, msg string) []byte {
	body := "SERROR\x00VERROR\x00C" + code + "\x00M" + msg + "\x00\x00"
	n := len(body) + 4
	return append([]byte{'E', byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}, body...)
}

// wireTags summarizes wire-protocol messages as their type tags, for ND_PGLITE_TRACE.
func wireTags(data []byte) string {
	var tags []string
	for len(data) >= 5 {
		tag := data[0]
		if tag == 0 && len(data) >= 8 && data[4] == 0 && data[5] == 3 { // untagged startup message
			tags = append(tags, "Startup")
			n := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
			if n <= 0 || n > len(data) {
				break
			}
			data = data[n:]
			continue
		}
		n := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
		tags = append(tags, string(tag))
		if n < 4 || n+1 > len(data) {
			tags = append(tags, "...")
			break
		}
		data = data[n+1:]
	}
	return strings.Join(tags, " ")
}
