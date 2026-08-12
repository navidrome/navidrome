package agents

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
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
	genres := seed.Tags[model.TagGenre]
	if len(genres) == 0 {
		return nil, nil
	}
	// Ask for extra so we can drop the seed itself and still fill the count.
	candidates, err := p.ds.MediaFile(ctx).GetAllByTags(model.TagGenre, genres, model.QueryOptions{
		Sort: "random",
		Max:  count + 1,
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
	songs := make([]Song, 0, len(mfs))
	for _, mf := range mfs {
		songs = append(songs, Song{Name: mf.Title, MBID: mf.MbzReleaseTrackID})
	}
	return songs
}

func init() {
	Register(LocalAgentName, localsConstructor)
}
