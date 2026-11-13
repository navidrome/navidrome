package configtest_test

import (
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfigtest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configtest Suite")
}

var _ = configtest.RestoreAfterEachSpec()
