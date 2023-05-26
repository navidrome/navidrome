package main

import (
	_ "net/http/pprof" //nolint:gosec

	"github.com/navidrome/navidrome/cmd"
)

func main() {
	cmd.Execute()
}
