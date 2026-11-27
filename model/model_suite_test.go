package model_test

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModel(t *testing.T) {
	tests.Init(t, true)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Model Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()
