//go:build tools

package main

import (
	_ "github.com/cespare/reflex"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "github.com/pressly/goose/cmd/goose"
	_ "golang.org/x/tools/cmd/goimports"
)
