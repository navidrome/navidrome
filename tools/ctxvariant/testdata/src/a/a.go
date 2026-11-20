package a

import (
	"context"
	"crypto/tls"
	"database/sql"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"

	"b"
)

func sqlCalls(ctx context.Context, db *sql.DB, tx *sql.Tx, stmt *sql.Stmt) {
	_, _ = db.Exec("q")                               // want `use \(\*database/sql\.DB\)\.ExecContext and pass the caller's ctx`
	_, _ = db.Query("q")                              // want `use \(\*database/sql\.DB\)\.QueryContext and pass the caller's ctx`
	_ = db.QueryRow("q")                              // want `use \(\*database/sql\.DB\)\.QueryRowContext and pass the caller's ctx`
	_, _ = db.Prepare("q")                            // want `use \(\*database/sql\.DB\)\.PrepareContext and pass the caller's ctx`
	_ = db.Ping()                                     // want `use \(\*database/sql\.DB\)\.PingContext and pass the caller's ctx`
	_, _ = tx.Exec("q")                               // want `use \(\*database/sql\.Tx\)\.ExecContext and pass the caller's ctx`
	_, _ = tx.Query("q")                              // want `use \(\*database/sql\.Tx\)\.QueryContext and pass the caller's ctx`
	_ = tx.QueryRow("q")                              // want `use \(\*database/sql\.Tx\)\.QueryRowContext and pass the caller's ctx`
	_, _ = tx.Prepare("q")                            // want `use \(\*database/sql\.Tx\)\.PrepareContext and pass the caller's ctx`
	_ = tx.Stmt(stmt)                                 // want `use \(\*database/sql\.Tx\)\.StmtContext and pass the caller's ctx`
	_, _ = stmt.Exec()                                // want `use \(\*database/sql\.Stmt\)\.ExecContext and pass the caller's ctx`
	_, _ = stmt.Query()                               // want `use \(\*database/sql\.Stmt\)\.QueryContext and pass the caller's ctx`
	_ = stmt.QueryRow()                               // want `use \(\*database/sql\.Stmt\)\.QueryRowContext and pass the caller's ctx`
	_, _ = db.ExecContext(ctx, "q")                   // already ctx-aware
	_, _ = db.BeginTx(ctx, nil)                       // already ctx-aware
	_, _ = db.Begin()                                 // want `use \(\*database/sql\.DB\)\.BeginTx and pass the caller's ctx`
	_, _ = embeddedDB{db}.Exec("q")                   // want `use \(\*database/sql\.DB\)\.ExecContext and pass the caller's ctx`
	_ = stmtFunc(stmt.Exec)                           // method value, not a call
	_, _ = http.NewRequest(http.MethodGet, "/", nil)  // want `use net/http\.NewRequestWithContext and pass the caller's ctx`
	_ = httptest.NewRequest(http.MethodGet, "/", nil) // want `use net/http/httptest\.NewRequestWithContext and pass the caller's ctx`
	_ = exec.Command("ls", "-l")                      // want `use os/exec\.CommandContext and pass the caller's ctx`
	_ = exec.CommandContext(ctx, "ls", "-l")          // already ctx-aware
}

type embeddedDB struct{ *sql.DB }

func stmtFunc(f func(...any) (sql.Result, error)) error { return nil }

type Repo struct{}

func (r *Repo) Foo(id string) error { return r.FooContext(context.TODO(), id) }

func (r *Repo) FooContext(ctx context.Context, id string) error { return nil }

// Bar's sibling takes an extra parameter, so it is not a drop-in replacement.
func (r *Repo) Bar(id string) error { return nil }

func (r *Repo) BarContext(ctx context.Context, id string, limit int) error { return nil }

// Baz's sibling does not take a ctx first.
func (r Repo) Baz(id string) error { return nil }

func (r Repo) BazContext(id string, ctx context.Context) error { return nil }

// Qux's sibling differs in variadic-ness.
func Qux(ids []string) {}

func QuxContext(ctx context.Context, ids ...string) {}

func Load(n int, names ...string) {}

func LoadWithContext(ctx context.Context, n int, names ...string) { Load(n, names...) }

// Recursion inside F's own body is not reported.
func Walk(n int) {
	if n > 0 {
		Walk(n - 1)
	}
}

func WalkContext(ctx context.Context, n int) { Walk(n) }

type Loader interface {
	Get(id string) error
	GetContext(ctx context.Context, id string) error
}

func local(r *Repo, v Repo, l Loader) {
	_ = r.Foo("1") // want `use \(\*a\.Repo\)\.FooContext and pass the caller's ctx`
	_ = v.Foo("1") // want `use \(\*a\.Repo\)\.FooContext and pass the caller's ctx`
	_ = r.Bar("1")
	_ = v.Baz("1")
	Qux(nil)
	Load(1, "a")         // want `use a\.LoadWithContext and pass the caller's ctx`
	_ = l.Get("1")       // want `use \(a\.Loader\)\.GetContext and pass the caller's ctx`
	func() { Load(2) }() // want `use a\.LoadWithContext and pass the caller's ctx`
}

var _ = func() bool {
	Load(3) // want `use a\.LoadWithContext and pass the caller's ctx`
	return true
}()

var _ = describe("spec", func() {
	_ = exec.Command("ls") // want `use os/exec\.CommandContext and pass the caller's ctx`
})

func describe(name string, body func()) bool { body(); return true }

type Box[T any] struct{ v T }

func (b Box[T]) Put(v T) {}

func (b Box[T]) PutContext(ctx context.Context, v T) {}

func Make[T any](v T) Box[T] { return Box[T]{v} }

func MakeContext[T any](ctx context.Context, v T) Box[T] { return Make(v) }

func generics() {
	b := Make(1) // want `use a\.MakeContext and pass the caller's ctx`
	b.Put(2)     // want `use \(a\.Box\[T\]\)\.PutContext and pass the caller's ctx`
}

func otherPackage() {
	b.Close("x") // want `use b\.CloseContext and pass the caller's ctx`
}

func irregular(c *http.Client) {
	_, _ = http.Get("/")               // want `use net/http\.NewRequestWithContext with \(\*net/http\.Client\)\.Do and pass the caller's ctx`
	_, _ = c.Post("/", "text", nil)    // want `use net/http\.NewRequestWithContext with \(\*net/http\.Client\)\.Do and pass the caller's ctx`
	_, _ = net.Dial("tcp", "x:1")      // want `use \(\*net\.Dialer\)\.DialContext and pass the caller's ctx`
	_, _ = net.Listen("tcp", ":0")     // want `use \(\*net\.ListenConfig\)\.Listen and pass the caller's ctx`
	_, _ = net.LookupHost("x")         // want `use \(\*net\.Resolver\)\.LookupHost and pass the caller's ctx`
	_, _ = tls.Dial("tcp", "x:1", nil) // want `use \(\*crypto/tls\.Dialer\)\.DialContext and pass the caller's ctx`
	_, _ = (&net.Dialer{}).DialContext(nil, "tcp", "x:1")
}

var _ = describe("irregular", func() {
	_, _ = http.Head("/") // want `use net/http\.NewRequestWithContext with \(\*net/http\.Client\)\.Do and pass the caller's ctx`
})

func Cmp[T any](v T) {}

func CmpContext[T comparable](ctx context.Context, v T) {}

func Loose[T comparable](v T) {}

func LooseContext[T any](ctx context.Context, v T) {}

func constraints() {
	Cmp([]int{1})
	Loose(1) // want `use a\.LooseContext and pass the caller's ctx`
}
