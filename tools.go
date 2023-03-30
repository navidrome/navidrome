//go:build tools

package main

import (
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "github.com/pressly/goose/cmd/goose"
	_ "golang.org/x/tools/cmd/goimports"
)
