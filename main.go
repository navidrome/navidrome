package main

import (
	_ "net/http/pprof" //nolint:gosec

	"github.com/navidrome/navidrome/cmd"
	"github.com/navidrome/navidrome/conf/buildtags"
)

//goland:noinspection GoBoolExpressions
func main() {
	// This import is used to force the inclusion of the `netgo` tag when compiling the project.
	// If you get errors like "undefined: buildtags.NETGO", this means you forgot to build the
	// project with the `netgo` build tag.
	_ = buildtags.NETGO

	cmd.Execute()
}
