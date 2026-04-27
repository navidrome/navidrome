package sonic

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const capabilitySonicSimilarity = "SonicSimilarity"

type SimilarResult struct {
	Song       agents.Song
	Similarity float64
}

type SimilarMatch struct {
	MediaFile  model.MediaFile
	Similarity float64
}

type Provider interface {
	GetSonicSimilarTracks(ctx context.Context, mf *model.MediaFile, count int) ([]SimilarResult, error)
	FindSonicPath(ctx context.Context, startMF, endMF *model.MediaFile, count int) ([]SimilarResult, error)
}

type PluginLoader interface {
	PluginNames(capability string) []string
	LoadSonicSimilarity(name string) (Provider, bool)
}

type Sonic struct {
	ds           model.DataStore
	pluginLoader PluginLoader
	matcher      *matcher.Matcher
}

func New(ds model.DataStore, pluginLoader PluginLoader, matcher *matcher.Matcher) *Sonic {
	return &Sonic{
		ds:           ds,
		pluginLoader: pluginLoader,
		matcher:      matcher,
	}
}

func (s *Sonic) HasProvider() bool {
	return len(s.pluginLoader.PluginNames(capabilitySonicSimilarity)) > 0
}

func (s *Sonic) loadProvider() (Provider, error) {
	names := s.pluginLoader.PluginNames(capabilitySonicSimilarity)
	if len(names) == 0 {
		return nil, model.ErrNotFound
	}
	provider, ok := s.pluginLoader.LoadSonicSimilarity(names[0])
	if !ok {
		return nil, model.ErrNotFound
	}
	return provider, nil
}

func (s *Sonic) resolveMatches(ctx context.Context, results []SimilarResult) ([]SimilarMatch, error) {
	songs := make([]agents.Song, len(results))
	for i, r := range results {
		songs[i] = r.Song
	}

	matchMap, err := s.matcher.MatchSongsIndexed(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("matching songs to library: %w", err)
	}

	var matches []SimilarMatch
	for i, r := range results {
		if mf, ok := matchMap[i]; ok {
			matches = append(matches, SimilarMatch{
				MediaFile:  mf,
				Similarity: r.Similarity,
			})
		}
	}
	return matches, nil
}

func (s *Sonic) GetSonicSimilarTracks(ctx context.Context, id string, count int) ([]SimilarMatch, error) {
	provider, err := s.loadProvider()
	if err != nil {
		return nil, err
	}

	mf, err := s.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, fmt.Errorf("getting media file %s: %w", id, err)
	}

	results, err := provider.GetSonicSimilarTracks(ctx, mf, count)
	if err != nil {
		log.Error(ctx, "Plugin GetSonicSimilarTracks failed", "id", id, err)
		return nil, err
	}

	return s.resolveMatches(ctx, results)
}

func (s *Sonic) FindSonicPath(ctx context.Context, startID, endID string, count int) ([]SimilarMatch, error) {
	provider, err := s.loadProvider()
	if err != nil {
		return nil, err
	}

	startMF, err := s.ds.MediaFile(ctx).Get(startID)
	if err != nil {
		return nil, fmt.Errorf("getting start media file %s: %w", startID, err)
	}
	endMF, err := s.ds.MediaFile(ctx).Get(endID)
	if err != nil {
		return nil, fmt.Errorf("getting end media file %s: %w", endID, err)
	}

	results, err := provider.FindSonicPath(ctx, startMF, endMF, count)
	if err != nil {
		log.Error(ctx, "Plugin FindSonicPath failed", "startId", startID, "endId", endID, err)
		return nil, err
	}

	return s.resolveMatches(ctx, results)
}
