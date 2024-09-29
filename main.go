package main

import (
	_ "net/http/pprof" //nolint:gosec

	"github.com/kardianos/service"
	"github.com/navidrome/navidrome/cmd"
)

func main() {

	if service.Interactive() {
		cmd.Execute()
	} else {
		prg := &cmd.SvcControl{}
		prg.Run()
	}
}
