// Test plugin for stream host functions and call time limits.
package main

//go:wasmexport spin
func spin() int32 {
	for {
	}
}

//go:wasmexport noop
func noop() int32 {
	return 0
}

func main() {}
