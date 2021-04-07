package main

import (
	"runtime"

	"github.com/navidrome/navidrome/cmd"
)

func main() {
	runtime.MemProfileRate = 0
	cmd.Execute()
}
