package nativeapi

import (
	"testing"

	"github.com/SunilSherkhane/navidrome/log"
	"github.com/SunilSherkhane/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNativeApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Native RESTful API Suite")
}
