// Test plugin for Navidrome sonic similarity integration tests.
// Build with: tinygo build -o ../test-sonic-similarity.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"errors"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/sonicsimilarity"
)

func init() {
	sonicsimilarity.Register(&testSonicSimilarity{})
}

type testSonicSimilarity struct{}

func checkConfigError() error {
	errMsg, hasErr := pdk.GetConfig("error")
	if !hasErr || errMsg == "" {
		return nil
	}
	return errors.New(errMsg)
}

func (t *testSonicSimilarity) GetSonicSimilarTracks(input sonicsimilarity.GetSonicSimilarTracksRequest) (sonicsimilarity.SonicSimilarityResponse, error) {
	if err := checkConfigError(); err != nil {
		return sonicsimilarity.SonicSimilarityResponse{}, err
	}
	count := int(input.Count)
	if count == 0 {
		count = 5
	}
	matches := make([]sonicsimilarity.SonicMatch, 0, count)
	for i := range count {
		matches = append(matches, sonicsimilarity.SonicMatch{
			Song: sonicsimilarity.SongRef{
				ID:     "similar-track-" + strconv.Itoa(i+1),
				Name:   "Similar to " + input.Song.Name + " #" + strconv.Itoa(i+1),
				Artist: input.Song.Artist,
			},
			Similarity: 1.0 - float64(i)*0.1,
		})
	}
	return sonicsimilarity.SonicSimilarityResponse{Matches: matches}, nil
}

func (t *testSonicSimilarity) FindSonicPath(input sonicsimilarity.FindSonicPathRequest) (sonicsimilarity.SonicSimilarityResponse, error) {
	if err := checkConfigError(); err != nil {
		return sonicsimilarity.SonicSimilarityResponse{}, err
	}
	count := int(input.Count)
	if count == 0 {
		count = 5
	}
	matches := make([]sonicsimilarity.SonicMatch, 0, count)
	for i := range count {
		matches = append(matches, sonicsimilarity.SonicMatch{
			Song: sonicsimilarity.SongRef{
				ID:     "path-track-" + strconv.Itoa(i+1),
				Name:   "Path " + input.StartSong.Name + " to " + input.EndSong.Name + " #" + strconv.Itoa(i+1),
				Artist: input.StartSong.Artist,
			},
			Similarity: 1.0 - float64(i)*0.05,
		})
	}
	return sonicsimilarity.SonicSimilarityResponse{Matches: matches}, nil
}

func main() {}
