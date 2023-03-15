package main

import (
	_ "net/http/pprof"

	"github.com/navidrome/navidrome/cmd"
)

func main() {
	cmd.Execute()
}
