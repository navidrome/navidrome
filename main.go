package main

import (
	"runtime"

	"github.com/navidrome/navidrome/cmd"
	_ "github.com/navidrome/navidrome/model/criteria"
)

func main() {
	runtime.MemProfileRate = 0
	cmd.Execute()
}
