// Test Matcher plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-matcher.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/types"
)

// TestMatcherInput is the input for the nd_test_matcher callback.
type TestMatcherInput struct {
	Songs    []types.SongRef `json:"songs"`
	Username string          `json:"username,omitempty"`
}

// TestMatcherOutput is the output from the nd_test_matcher callback.
// MatchedIDs and Starred are aligned to the input: an empty string at index i
// means no match; Starred[i] reflects the matched track's starred flag.
type TestMatcherOutput struct {
	MatchedIDs []string `json:"matched_ids"`
	Starred    []bool   `json:"starred"`
	Error      *string  `json:"error,omitempty"`
}

// nd_test_matcher forwards the input song list to the host matcher and returns matched track IDs.
//
//go:wasmexport nd_test_matcher
func ndTestMatcher() int32 {
	var input TestMatcherInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestMatcherOutput{Error: &errStr})
		return 0
	}

	results, err := host.MatcherMatchSongs(input.Songs, host.MatchOptions{Username: input.Username})
	if err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestMatcherOutput{Error: &errStr})
		return 0
	}

	ids := make([]string, len(results))
	starred := make([]bool, len(results))
	for i, t := range results {
		if t != nil {
			ids[i] = t.ID
			starred[i] = t.Starred
		}
	}
	pdk.OutputJSON(TestMatcherOutput{MatchedIDs: ids, Starred: starred})
	return 0
}

func main() {}
