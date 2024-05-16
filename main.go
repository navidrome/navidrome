package main

import (
	_ "net/http/pprof" //nolint:gosec

	_ "github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/cmd"
)

func main() {
	cmd.Execute()
}
