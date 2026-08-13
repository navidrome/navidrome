package external

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/random"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/sync/errgroup"
)

const (
	maxSeeds = 5
	// Subsonic passes the client's count through unbounded, and it ends up as a SQL limit. 500 is
	// what the widest caller (similarAlbums, limit*5) legitimately asks for.
	maxSimilarSongs = 500
)

func (e *provider) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	// Subsonic passes the client's count straight through: a non-positive one has no valid
	// interpretation, and an enormous one overflows the +1 in the local agent's query limit.
	if count <= 0 {
		return nil, nil
	}
	count = min(count, maxSimilarSongs)
	entity, err := model.GetEntityByID(ctx, e.ds, id)
	if err != nil {
		// Genre ids don't resolve via GetEntityByID; look them up before giving up.
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
		genre, err := e.ds.Genre(ctx).Get(id)
		if err != nil {
			return nil, err
		}
		return e.seedMix(ctx, count, func() (model.MediaFiles, error) {
			return e.sampleGenreTracks(ctx, genre, maxSeeds)
		})
	}

	// Try entity-specific similarity first, then fall back to seed-track sampling.
	switch v := entity.(type) {
	case *model.MediaFile:
		return e.mixFromAgent(ctx, count,
			func() ([]agents.Song, error) {
				return e.ag.GetSimilarSongsByTrack(ctx, v.ID, v.Title, v.Artist, v.MbzRecordingID, count)
			},
			func() (model.MediaFiles, error) {
				return e.similarSongsFallback(ctx, id, count)
			})
	case *model.Album:
		return e.mixFromAgent(ctx, count,
			func() ([]agents.Song, error) {
				return e.ag.GetSimilarSongsByAlbum(ctx, v.ID, v.Name, v.AlbumArtist, v.MbzAlbumID, count)
			},
			func() (model.MediaFiles, error) {
				return e.seedMix(ctx, count, func() (model.MediaFiles, error) {
					return e.sampleAlbumTracks(ctx, v.ID, maxSeeds)
				})
			})
	case *model.Artist:
		return e.mixFromAgent(ctx, count,
			func() ([]agents.Song, error) {
				return e.ag.GetSimilarSongsByArtist(ctx, v.ID, v.Name, v.MbzArtistID, count)
			},
			func() (model.MediaFiles, error) {
				return e.similarSongsFallback(ctx, id, count)
			},
			func() (model.MediaFiles, error) {
				return e.seedMix(ctx, count, func() (model.MediaFiles, error) {
					return e.sampleArtistTracks(ctx, v.ID, maxSeeds)
				})
			})
	case *model.Playlist:
		return e.seedMix(ctx, count, func() (model.MediaFiles, error) {
			return e.samplePlaylistTracks(ctx, v.ID, maxSeeds)
		})
	default:
		log.Warn(ctx, "Unknown entity type", "id", id, "type", fmt.Sprintf("%T", entity))
		return nil, model.ErrNotFound
	}
}

// mixFromAgent returns the agent's recommendations matched to library tracks, topped up from the
// fallbacks in order when they don't fill the mix on their own.
func (e *provider) mixFromAgent(ctx context.Context, count int, fetch func() ([]agents.Song, error), fallbacks ...func() (model.MediaFiles, error)) (model.MediaFiles, error) {
	var matched model.MediaFiles
	if songs, err := fetch(); err == nil {
		var merr error
		if matched, merr = e.matcher.MatchSongs(ctx, songs, count); merr != nil {
			return nil, merr
		}
	}
	return topUp(ctx, matched, count, fallbacks...)
}

// topUp draws on each source in turn until the mix holds count tracks. A short mix is worse than a
// slow one: clients that ask for count and get a handful re-ask with an ever-larger limit forever.
// Sources are measured against the whole mix, so a later (costlier) one is skipped once it's full.
func topUp(ctx context.Context, res model.MediaFiles, count int, sources ...func() (model.MediaFiles, error)) (model.MediaFiles, error) {
	// Dedup before measuring: the matcher re-emits a track when the same input song repeats, so a
	// full-looking res can hold fewer unique tracks than count and stop the top-up too early.
	res = dedupByID(res)
	var firstErr error
	for _, more := range sources {
		if len(res) >= count {
			break
		}
		extra, err := more()
		if err != nil {
			log.Debug(ctx, "Could not top up a short mix", "have", len(res), "want", count, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		res = dedupByID(append(res, extra...))
	}
	if len(res) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return res[:min(len(res), count)], nil
}

// seedMix samples seed tracks, runs each through the agent chain's per-track similarity and merges
// the results, falling back to the seeds themselves so the result is never empty.
func (e *provider) seedMix(ctx context.Context, count int, sample func() (model.MediaFiles, error)) (model.MediaFiles, error) {
	seeds, err := sample()
	if err != nil {
		return nil, err
	}
	if len(seeds) == 0 {
		return nil, nil
	}
	seeds = seeds[:min(len(seeds), maxSeeds)]

	// The per-seed similarity calls are independent and hit the (possibly remote) agent chain, so
	// run them concurrently. Best-effort: a seed that errors just contributes nothing.
	perSeed := make([][]agents.Song, len(seeds))
	var g errgroup.Group
	for i, seed := range seeds {
		g.Go(func() error {
			if s, err := e.ag.GetSimilarSongsByTrack(ctx, seed.ID, seed.Title, seed.Artist, seed.MbzRecordingID, count); err == nil {
				perSeed[i] = s
			}
			return nil
		})
	}
	_ = g.Wait()

	var songs []agents.Song
	for _, s := range perSeed {
		songs = append(songs, s...)
	}
	// Match the whole merged set, not just count of it: the matcher re-emits a track when two
	// seeds recommend it identically, so the duplicates have to be dropped before trimming. Every
	// seed reaches the shuffle, so no seed can crowd out the others.
	matched, err := e.matcher.MatchSongs(ctx, songs, len(songs))
	if err != nil {
		return nil, err
	}
	matched = dedupByID(matched)
	if len(matched) == 0 {
		matched = seeds
	}
	rand.Shuffle(len(matched), func(i, j int) { matched[i], matched[j] = matched[j], matched[i] })
	if len(matched) > count {
		matched = matched[:count]
	}
	return matched, nil
}

func (e *provider) samplePlaylistTracks(ctx context.Context, playlistID string, n int) (model.MediaFiles, error) {
	// Refresh: a smart playlist materializes no tracks until it is evaluated, so skipping it would
	// mix an empty seed set. It is a no-op for regular playlists and inside the refresh delay.
	repo := e.ds.Playlist(ctx).Tracks(playlistID, true)
	if repo == nil {
		return nil, model.ErrNotFound
	}
	// A playlist can hold the same file at several positions, so over-fetch and dedup: a repeated
	// seed wastes an agent call and can reach the mix twice through the seed fallback.
	tracks, err := repo.GetAll(model.QueryOptions{
		Sort:    "random",
		Max:     n * 4,
		Filters: squirrel.Eq{"missing": false},
	})
	if err != nil {
		return nil, err
	}
	mfs := dedupByID(tracks.MediaFiles())
	return mfs[:min(len(mfs), n)], nil
}

func dedupByID(mfs model.MediaFiles) model.MediaFiles {
	seen := make(map[string]struct{}, len(mfs))
	return slice.Filter(mfs, func(mf model.MediaFile) bool {
		if _, dup := seen[mf.ID]; dup {
			return false
		}
		seen[mf.ID] = struct{}{}
		return true
	})
}

func (e *provider) sampleAlbumTracks(ctx context.Context, albumID string, n int) (model.MediaFiles, error) {
	return e.sampleTracks(ctx, squirrel.Eq{"album_id": albumID}, n)
}

func (e *provider) sampleArtistTracks(ctx context.Context, artistID string, n int) (model.MediaFiles, error) {
	// media_file.artist_id is the deprecated primary artist, so it misses an artist credited only
	// on the album, as on compilations. Same filter the artist listings use.
	filter := persistence.ParticipantIDFilter("media_file", artistID, model.RoleArtist, model.RoleAlbumArtist)
	return e.sampleTracks(ctx, filter, n)
}

func (e *provider) sampleGenreTracks(ctx context.Context, genre *model.Genre, n int) (model.MediaFiles, error) {
	return e.sampleTracks(ctx, persistence.SongGenres.ByID(genre.ID), n)
}

// sampleTracks returns up to n random present tracks. Seeds can end up in the mix verbatim, so
// missing files would surface as unplayable entries.
func (e *provider) sampleTracks(ctx context.Context, filter squirrel.Sqlizer, n int) (model.MediaFiles, error) {
	return e.ds.MediaFile(ctx).GetRandom(model.QueryOptions{
		Filters: squirrel.And{filter, squirrel.Eq{"missing": false}},
		Max:     n,
	})
}

// similarSongsFallback uses the original similar artists + top songs algorithm. The idea is to
// get the artist of the given entity, retrieve similar artists, get their top songs, and pick
// a weighted random selection of songs to return as similar songs.
func (e *provider) similarSongsFallback(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	e.callGetSimilarArtists(ctx, e.ag, &artist, 15, false)
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	weightedSongs := random.NewWeightedChooser[model.MediaFile]()
	addArtist := func(a model.Artist, weightedSongs *random.WeightedChooser[model.MediaFile], count, artistWeight int) error {
		if utils.IsCtxDone(ctx) {
			log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
			return ctx.Err()
		}

		topCount := max(count, 20)
		topSongs, err := e.getMatchingTopSongs(ctx, e.ag, &auxArtist{Artist: a}, topCount)
		if err != nil {
			log.Warn(ctx, "Error getting artist's top songs", "artist", a.Name, err)
			return nil
		}

		weight := topCount * (4 + artistWeight)
		for _, mf := range topSongs {
			weightedSongs.Add(mf, weight)
			weight -= 4
		}
		return nil
	}

	err = addArtist(artist.Artist, weightedSongs, count, 10)
	if err != nil {
		return nil, err
	}
	for _, a := range artist.SimilarArtists {
		err := addArtist(a, weightedSongs, count, 0)
		if err != nil {
			return nil, err
		}
	}

	var similarSongs model.MediaFiles
	for len(similarSongs) < count && weightedSongs.Size() > 0 {
		s, err := weightedSongs.Pick()
		if err != nil {
			log.Warn(ctx, "Error getting weighted song", err)
			continue
		}
		similarSongs = append(similarSongs, s)
	}

	return similarSongs, nil
}
