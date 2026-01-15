//go:build tools

// This file ensures the extism/go-pdk dependency stays in go.mod.
// The PDK parser loads this package at runtime using go/packages.
// Without this import, `go mod tidy` would remove it since it's not directly imported elsewhere.
package main

import _ "github.com/extism/go-pdk"
