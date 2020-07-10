package core

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/tests"
	"github.com/djherbis/fscache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEngine(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Suite")
}

var testCache fscache.Cache
var testCacheDir string

var _ = Describe("Core Suite Setup", func() {
	BeforeSuite(func() {
		testCacheDir, _ = ioutil.TempDir("", "core_test_cache")
		fs, _ := fscache.NewFs(testCacheDir, 0755)
		testCache, _ = fscache.NewCache(fs, nil)
	})

	AfterSuite(func() {
		os.RemoveAll(testCacheDir)
	})
})
