package main

import (
	_ "net/http/pprof" //nolint:gosec

	"github.com/navidrome/navidrome/cmd"
	"github.com/navidrome/navidrome/conf/buildtags"
)

//goland:noinspection GoBoolExpressions
func main() {
	// These references force the inclusion of build tags when compiling the project.
	// If you get compilation errors like "undefined: buildtags.NETGO", this means you forgot to specify
	// the required build tags when compiling the project.
	// To avoid these kind of errors, you should use `make build` to compile the project.
	_ = buildtags.NETGO
	_ = buildtags.SQLITE_FTS5

	cmd.Execute()
}
