package ctxvariant_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/navidrome/navidrome/tools/ctxvariant"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxvariant.Analyzer, "a")
}
