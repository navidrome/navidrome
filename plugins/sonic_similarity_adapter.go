package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const CapabilitySonicSimilarity Capability = "SonicSimilarity"

const (
	FuncGetSonicSimilarTracks = "nd_get_sonic_similar_tracks"
	FuncFindSonicPath         = "nd_find_sonic_path"
)

func init() {
	registerCapability(
		CapabilitySonicSimilarity,
		FuncGetSonicSimilarTracks,
		FuncFindSonicPath,
	)
}

func newSonicSimilarityPlugin(p *plugin) *SonicSimilarityPlugin {
	return &SonicSimilarityPlugin{name: p.name, plugin: p}
}

type SonicSimilarityPlugin struct {
	name   string
	plugin *plugin
}

func (a *SonicSimilarityPlugin) GetSonicSimilarTracks(ctx context.Context, mf *model.MediaFile, count int) ([]sonic.SimilarResult, error) {
	req := capabilities.GetSonicSimilarTracksRequest{
		Song:  mediaFileToSongRef(mf),
		Count: int32(count),
	}
	resp, err := callPluginFunction[capabilities.GetSonicSimilarTracksRequest, capabilities.SonicSimilarityResponse](
		ctx, a.plugin, FuncGetSonicSimilarTracks, req,
	)
	if err != nil {
		return nil, err
	}
	return sonicMatchesToSimilarResults(resp.Matches), nil
}

func (a *SonicSimilarityPlugin) FindSonicPath(ctx context.Context, startMf, endMf *model.MediaFile, count int) ([]sonic.SimilarResult, error) {
	req := capabilities.FindSonicPathRequest{
		StartSong: mediaFileToSongRef(startMf),
		EndSong:   mediaFileToSongRef(endMf),
		Count:     int32(count),
	}
	resp, err := callPluginFunction[capabilities.FindSonicPathRequest, capabilities.SonicSimilarityResponse](
		ctx, a.plugin, FuncFindSonicPath, req,
	)
	if err != nil {
		return nil, err
	}
	return sonicMatchesToSimilarResults(resp.Matches), nil
}

func mediaFileToSongRef(mf *model.MediaFile) capabilities.SongRef {
	ref := capabilities.SongRef{
		ID:         mf.ID,
		Name:       mf.Title,
		MBID:       mf.MbzRecordingID,
		Artist:     mf.Artist,
		ArtistMBID: mf.MbzArtistID,
		Album:      mf.Album,
		AlbumMBID:  mf.MbzAlbumID,
		Duration:   mf.Duration,
	}
	if isrcs := mf.Tags.Values(model.TagISRC); len(isrcs) > 0 {
		ref.ISRC = isrcs[0]
	}
	return ref
}

func sonicMatchesToSimilarResults(matches []capabilities.SonicMatch) []sonic.SimilarResult {
	results := make([]sonic.SimilarResult, len(matches))
	for i, m := range matches {
		results[i] = sonic.SimilarResult{
			Song:       songRefToAgentSong(m.Song),
			Similarity: m.Similarity,
		}
	}
	return results
}

var _ sonic.Provider = (*SonicSimilarityPlugin)(nil)
