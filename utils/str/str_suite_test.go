package str_test

import (
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStrClear(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Str Suite")
}

var _ = configtest.RestoreAfterEachSpec()
