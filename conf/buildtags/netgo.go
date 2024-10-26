//go:build netgo

package buildtags

// NOTICE: This file was created to force the inclusion of the `netgo` tag when compiling the project.
// If the tag is not included, the compilation will fail because this variable won't be defined, and the `main.go`
// file requires it.

// Why this tag is required? See https://github.com/navidrome/navidrome/issues/700

var NETGO = true
