// Package buildtags provides compile-time enforcement of required build tags.
//
// Each file in this package is guarded by a build constraint and exports a variable
// that main.go references. If a required tag is missing during compilation, the build
// fails with an "undefined" error, directing the developer to use `make build`.
package buildtags
