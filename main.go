package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/cmd"
	_ "github.com/navidrome/navidrome/model/criteria"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.MemProfileRate = 0
	cmd.Execute()
}
