package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHostgen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hostgen CLI Suite")
}
