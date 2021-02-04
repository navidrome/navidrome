package main

import (
	"runtime"

	"github.com/deluan/navidrome/cmd"
)

func main() {
	runtime.MemProfileRate = 0
	cmd.Execute()
}
