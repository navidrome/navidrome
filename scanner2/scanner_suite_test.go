package scanner2_test

import (
	"testing"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScanner(t *testing.T) {
	tests.Init(t, true)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}
