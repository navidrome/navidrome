package criteria

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestCriteria(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "Criteria Suite")
}
