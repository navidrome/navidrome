package responses

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/pmezard/go-difflib/difflib"
)

func TestSubsonicApiResponses(t *testing.T) {
	log.SetLevel(log.LevelError)
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Subsonic API Responses Suite")
}

func MatchSnapshot() types.GomegaMatcher {
	return &snapshotMatcher{}
}

type snapshotMatcher struct {
	diff string
}

func (matcher *snapshotMatcher) Match(actual any) (success bool, err error) {
	actualBytes, ok := actual.([]byte)
	if !ok {
		return false, fmt.Errorf("MatchSnapshot expects []byte, got %T", actual)
	}

	name := ginkgo.CurrentSpecReport().FullText()
	actualStr := strings.TrimSpace(string(actualBytes))

	snapshotPath := filepath.Join(".snapshots", name)

	// Support UPDATE_SNAPSHOTS=true for `make snapshots`
	if _, update := os.LookupEnv("UPDATE_SNAPSHOTS"); update {
		err := os.MkdirAll(".snapshots", os.ModePerm)
		if err != nil {
			return false, err
		}
		return true, os.WriteFile(snapshotPath, []byte(actualStr+"\n"), 0600)
	}

	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		return false, err
	}

	// Normalize line endings so snapshots work on Windows (CRLF) and Unix (LF)
	normalizedExpected := strings.ReplaceAll(string(expected), "\r\n", "\n")
	normalizedExpected = strings.TrimSpace(normalizedExpected)

	if actualStr == normalizedExpected {
		return true, nil
	}

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(normalizedExpected),
		B:        difflib.SplitLines(actualStr),
		FromFile: "snapshot",
		ToFile:   "actual",
		Context:  3,
	})
	matcher.diff = diff
	return false, nil
}

func (matcher *snapshotMatcher) FailureMessage(_ any) (message string) {
	return "Expected to match saved snapshot\n" + matcher.diff
}

func (matcher *snapshotMatcher) NegatedFailureMessage(_ any) (message string) {
	return "Expected to not match saved snapshot\n"
}
