package responses

import (
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func TestSubsonicApiResponses(t *testing.T) {
	log.SetLevel(log.LevelError)
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Subsonic API Responses Suite")
}

func MatchSnapshot() types.GomegaMatcher {
	c := cupaloy.New(cupaloy.FailOnUpdate(false))
	return &snapshotMatcher{c}
}

type snapshotMatcher struct {
	c *cupaloy.Config
}

func (matcher snapshotMatcher) Match(actual interface{}) (success bool, err error) {
	actualJson := strings.TrimSpace(string(actual.([]byte)))
	err = matcher.c.SnapshotWithName(ginkgo.CurrentSpecReport().FullText(), actualJson)
	success = err == nil
	return
}

func (matcher snapshotMatcher) FailureMessage(_ interface{}) (message string) {
	return "Expected to match saved snapshot\n"
}

func (matcher snapshotMatcher) NegatedFailureMessage(_ interface{}) (message string) {
	return "Expected to not match saved snapshot\n"
}
