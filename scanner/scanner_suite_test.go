package scanner

import (
	"testing"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO Fix OS dependencies
func xTestScanner(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}

var _ = Describe("TODO: REMOVE", func() {
	conf.Sonic.DbPath = "./testDB"
	log.SetLevel(log.LevelDebug)
	ds := persistence.New()
	It("WORKS!", func() {
		t := NewTagScanner("/Users/deluan/Music/iTunes/iTunes Media/Music", ds)
		//t := NewTagScanner("/Users/deluan/Development/cloudsonic/sonic-server/tests/fixtures", ds)
		Expect(t.Scan(nil, time.Time{})).To(BeNil())
	})
})
