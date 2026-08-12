package agents

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/slice"
)

const LocalAgentName = "local"

type localAgent struct {
	ds model.DataStore
}

func localsConstructor(ds model.DataStore) Interface {
	return &localAgent{ds}
}

func (p *localAgent) AgentName() string {
	return LocalAgentName
}

func (p *localAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	top, err := p.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Sort:  "playCount",
		Order: "desc",
		Max:   count,
		Filters: squirrel.And{
			squirrel.Eq{"artist_id": id},
			squirrel.Or{
				squirrel.Eq{"starred": true},
				squirrel.Eq{"rating": 5},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return songsFrom(top), nil
}

func (p *localAgent) GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error) {
	seed, err := p.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}
	// Tag ids derive from (name, value), so the seed's genre ids need no extra query.
	genreIDs := slice.Map(seed.Tags.Flatten(model.TagGenre), func(t model.Tag) string { return t.ID })
	if len(genreIDs) == 0 {
		return nil, nil
	}
	// Ask for extra so we can drop the seed itself and still fill the count.
	candidates, err := p.ds.MediaFile(ctx).GetRandom(model.QueryOptions{
		Filters: persistence.SongGenres.ByID(genreIDs),
		Max:     count + 1,
	})
	if err != nil {
		return nil, err
	}
	filtered := make(model.MediaFiles, 0, len(candidates))
	for _, s := range candidates {
		if s.ID == id {
			continue
		}
		filtered = append(filtered, s)
		if len(filtered) >= count {
			break
		}
	}
	return songsFrom(filtered), nil
}

func songsFrom(mfs model.MediaFiles) []Song {
	if len(mfs) == 0 {
		return nil
	}

	// These are library tracks already, so carry the id: the matcher resolves by id first and
	// exactly. MBID must be the recording id — the matcher matches mbz_recording_id.
	return slice.Map(mfs, func(mf model.MediaFile) Song {
		return Song{ID: mf.ID, Name: mf.Title, MBID: mf.MbzRecordingID}
	})
}

func init() {
	Register(LocalAgentName, localsConstructor)
}
