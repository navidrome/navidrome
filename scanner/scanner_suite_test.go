package scanner

import (
	"testing"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO Fix OS dependencies
func xTestScanner(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}

var _ = XDescribe("TODO: REMOVE", func() {
	It("WORKS!", func() {
		conf.Server.DbPath = "./testDB"
		log.SetLevel(log.LevelDebug)
		ds := persistence.New()

		t := NewTagScanner("/Users/deluan/Music/iTunes/iTunes Media/Music", ds)
		//t := NewTagScanner("/Users/deluan/Development/navidrome/navidrome/tests/fixtures", ds)
		Expect(t.Scan(nil, time.Time{})).To(BeNil())
	})
})
