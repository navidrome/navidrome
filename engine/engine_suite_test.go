package engine

import (
	"testing"

	"github.com/cloudsonic/sonic-server/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}
