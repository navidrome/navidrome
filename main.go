package main

import (
	_ "net/http/pprof" //nolint:gosec

	"github.com/navidrome/navidrome/cmd"
	"github.com/navidrome/navidrome/conf/buildtags"
)

//goland:noinspection GoBoolExpressions
func main() {
	// This import is used to force the inclusion of the `netgo` tag when compiling the project.
	// If you get compilation errors like "undefined: buildtags.NETGO", this means you forgot to specify
	// the `netgo` build tag when compiling the project.
	// To avoid these kind of errors, you should use `make build` to compile the project.
	_ = buildtags.NETGO

	cmd.Execute()
}
