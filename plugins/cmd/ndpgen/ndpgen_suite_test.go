package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNdpgen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NDPGen CLI Suite")
}
