package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/cmd"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.MemProfileRate = 0
	cmd.Execute()
}
