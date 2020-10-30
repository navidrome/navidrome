package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core/lastfm"
	"github.com/deluan/navidrome/core/spotify"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/microcosm-cc/bluemonday"
	"github.com/xrash/smetrics"
)

const placeholderArtistImageSmallUrl = "https://lastfm.freetls.fastly.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageMediumUrl = "https://lastfm.freetls.fastly.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageLargeUrl = "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png"

type ExternalInfo interface {
	UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error)
	SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error)
	TopSongs(ctx context.Context, artist string, count int) (model.MediaFiles, error)
}

func NewExternalInfo(ds model.DataStore, lfm *lastfm.Client, spf *spotify.Client) ExternalInfo {
	return &externalInfo{ds: ds, lfm: lfm, spf: spf}
}

type externalInfo struct {
	ds  model.DataStore
	lfm *lastfm.Client
	spf *spotify.Client
}

const UnavailableArtistID = "-1"

func (e *externalInfo) UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	// If we have updated info, just return it
	if time.Since(artist.ExternalInfoUpdatedAt) < consts.ArtistInfoTimeToLive {
		log.Debug("Found cached ArtistInfo", "updatedAt", artist.ExternalInfoUpdatedAt, "name", artist.Name)
		err := e.loadSimilar(ctx, artist, includeNotPresent)
		return artist, err
	}
	log.Debug("ArtistInfo not cached", "updatedAt", artist.ExternalInfoUpdatedAt, "id", id)

	// TODO Load from local: artist.jpg/png/webp, artist.json (with the remaining info)

	var wg sync.WaitGroup
	e.callArtistInfo(ctx, artist, &wg)
	e.callArtistImages(ctx, artist, &wg)
	e.callSimilarArtists(ctx, artist, count, &wg)
	wg.Wait()

	// Use placeholders if could not get from external sources
	e.setBio(artist, "Biography not available")
	e.setSmallImageUrl(artist, placeholderArtistImageSmallUrl)
	e.setMediumImageUrl(artist, placeholderArtistImageMediumUrl)
	e.setLargeImageUrl(artist, placeholderArtistImageLargeUrl)

	artist.ExternalInfoUpdatedAt = time.Now()
	err = e.ds.Artist(ctx).Put(artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artistImageUrl", "id", id, err)
	}

	if !includeNotPresent {
		similar := artist.SimilarArtists
		artist.SimilarArtists = nil
		for _, s := range similar {
			if s.ID == UnavailableArtistID {
				continue
			}
			artist.SimilarArtists = append(artist.SimilarArtists, s)
		}
	}

	log.Trace(ctx, "ArtistInfo collected", "artist", artist)

	return artist, nil
}

func (e *externalInfo) getArtist(ctx context.Context, id string) (*model.Artist, error) {
	var entity interface{}
	entity, err := GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return nil, err
	}

	switch v := entity.(type) {
	case *model.Artist:
		return v, nil
	case *model.MediaFile:
		return e.ds.Artist(ctx).Get(v.ArtistID)
	case *model.Album:
		return e.ds.Artist(ctx).Get(v.AlbumArtistID)
	}
	return nil, model.ErrNotFound
}

// Replace some Unicode chars with their equivalent ASCII
func clearName(name string) string {
	name = strings.ReplaceAll(name, "–", "-")
	name = strings.ReplaceAll(name, "‐", "-")
	name = strings.ReplaceAll(name, "“", `"`)
	name = strings.ReplaceAll(name, "”", `"`)
	name = strings.ReplaceAll(name, "‘", `'`)
	name = strings.ReplaceAll(name, "’", `'`)
	return name
}

func (e *externalInfo) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	if e.lfm == nil {
		log.Warn(ctx, "Last.FM client not configured")
		return nil, model.ErrNotAvailable
	}

	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	artists, err := e.similarArtists(ctx, clearName(artist.Name), count, false)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(artists)+1)
	ids[0] = artist.ID
	for i, a := range artists {
		ids[i+1] = a.ID
	}

	return e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"artist_id": ids},
		Max:     count,
		Sort:    "random()",
	})
}

func (e *externalInfo) similarArtists(ctx context.Context, artistName string, count int, includeNotPresent bool) (model.Artists, error) {
	var result model.Artists
	var notPresent []string

	log.Debug(ctx, "Calling Last.FM ArtistGetSimilar", "artist", artistName)
	similar, err := e.lfm.ArtistGetSimilar(ctx, artistName, count)
	if err != nil {
		return nil, err
	}

	// First select artists that are present.
	for _, s := range similar {
		sa, err := e.findArtistByName(ctx, s.Name)
		if err != nil {
			notPresent = append(notPresent, s.Name)
			continue
		}
		result = append(result, *sa)
	}

	// Then fill up with non-present artists
	if includeNotPresent {
		for _, s := range notPresent {
			sa := model.Artist{ID: UnavailableArtistID, Name: s}
			result = append(result, sa)
		}
	}

	return result, nil
}

func (e *externalInfo) findArtistByName(ctx context.Context, artistName string) (*model.Artist, error) {
	artists, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Like{"name": artistName},
		Max:     1,
	})
	if err != nil {
		return nil, err
	}
	if len(artists) == 0 {
		return nil, model.ErrNotFound
	}
	return &artists[0], nil
}

func (e *externalInfo) TopSongs(ctx context.Context, artistName string, count int) (model.MediaFiles, error) {
	if e.lfm == nil {
		log.Warn(ctx, "Last.FM client not configured")
		return nil, model.ErrNotAvailable
	}
	artist, err := e.findArtistByName(ctx, artistName)
	if err != nil {
		log.Error(ctx, "Artist not found", "name", artistName, err)
		return nil, nil
	}
	artistName = clearName(artistName)

	log.Debug(ctx, "Calling Last.FM ArtistGetTopTracks", "artist", artistName, "id", artist.ID)
	tracks, err := e.lfm.ArtistGetTopTracks(ctx, artistName, count)
	if err != nil {
		return nil, err
	}
	var songs model.MediaFiles
	for _, t := range tracks {
		mf, err := e.findMatchingTrack(ctx, t.MBID, artist.ID, t.Name)
		if err != nil {
			continue
		}
		songs = append(songs, *mf)
	}
	return songs, nil
}

func (e *externalInfo) findMatchingTrack(ctx context.Context, mbid string, artistID, title string) (*model.MediaFile, error) {
	if mbid != "" {
		mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"mbz_track_id": mbid},
		})
		if err == nil && len(mfs) > 0 {
			return &mfs[0], nil
		}
	}
	mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Or{
				squirrel.Eq{"artist_id": artistID},
				squirrel.Eq{"album_artist_id": artistID},
			},
			squirrel.Like{"title": title},
		},
		Sort: "starred desc, rating desc, year asc",
	})
	if err != nil || len(mfs) == 0 {
		return nil, model.ErrNotFound
	}
	return &mfs[0], nil
}

func (e *externalInfo) callArtistInfo(ctx context.Context, artist *model.Artist, wg *sync.WaitGroup) {
	if e.lfm != nil {
		name := clearName(artist.Name)
		log.Debug(ctx, "Calling Last.FM ArtistGetInfo", "artist", name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()
			lfmArtist, err := e.lfm.ArtistGetInfo(ctx, name)
			if err != nil {
				log.Error(ctx, "Error calling Last.FM", "artist", name, err)
			} else {
				log.Debug(ctx, "Got info from Last.FM", "artist", name, "info", lfmArtist.Bio.Summary, "elapsed", time.Since(start))
			}
			e.setBio(artist, lfmArtist.Bio.Summary)
			e.setExternalUrl(artist, lfmArtist.URL)
			e.setMbzID(artist, lfmArtist.MBID)
		}()
	}
}

func (e *externalInfo) searchArtist(ctx context.Context, name string) (*spotify.Artist, error) {
	artists, err := e.spf.SearchArtists(ctx, name, 40)
	if err != nil || len(artists) == 0 {
		return nil, model.ErrNotFound
	}
	name = strings.ToLower(name)

	// Sort results, prioritizing artists with images, with similar names and with high popularity, in this order
	sort.Slice(artists, func(i, j int) bool {
		ai := fmt.Sprintf("%-5t-%03d-%04d", len(artists[i].Images) == 0, smetrics.WagnerFischer(name, strings.ToLower(artists[i].Name), 1, 1, 2), 1000-artists[i].Popularity)
		aj := fmt.Sprintf("%-5t-%03d-%04d", len(artists[j].Images) == 0, smetrics.WagnerFischer(name, strings.ToLower(artists[j].Name), 1, 1, 2), 1000-artists[j].Popularity)
		return strings.Compare(ai, aj) < 0
	})

	// If the first one has the same name, that's the one
	if strings.ToLower(artists[0].Name) != name {
		return nil, model.ErrNotFound
	}
	return &artists[0], err
}

func (e *externalInfo) callSimilarArtists(ctx context.Context, artist *model.Artist, count int, wg *sync.WaitGroup) {
	if e.lfm != nil {
		name := clearName(artist.Name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()
			similar, err := e.similarArtists(ctx, name, count, true)
			if err != nil {
				log.Error(ctx, "Error calling Last.FM", "artist", name, err)
				return
			}
			log.Debug(ctx, "Got similar artists from Last.FM", "artist", name, "info", "elapsed", time.Since(start))
			artist.SimilarArtists = similar
		}()
	}
}

func (e *externalInfo) callArtistImages(ctx context.Context, artist *model.Artist, wg *sync.WaitGroup) {
	if e.spf != nil {
		name := clearName(artist.Name)
		log.Debug(ctx, "Calling Spotify SearchArtist", "artist", name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()

			a, err := e.searchArtist(ctx, name)
			if err != nil {
				if err == model.ErrNotFound {
					log.Warn(ctx, "Artist not found in Spotify", "artist", name)
				} else {
					log.Error(ctx, "Error calling Spotify", "artist", name, err)
				}
				return
			}
			spfImages := a.Images
			log.Debug(ctx, "Got images from Spotify", "artist", name, "images", spfImages, "elapsed", time.Since(start))

			sort.Slice(spfImages, func(i, j int) bool { return spfImages[i].Width > spfImages[j].Width })
			if len(spfImages) >= 1 {
				e.setLargeImageUrl(artist, spfImages[0].URL)
			}
			if len(spfImages) >= 2 {
				e.setMediumImageUrl(artist, spfImages[1].URL)
			}
			if len(spfImages) >= 3 {
				e.setSmallImageUrl(artist, spfImages[2].URL)
			}
		}()
	}
}

func (e *externalInfo) setBio(artist *model.Artist, bio string) {
	policy := bluemonday.UGCPolicy()
	if artist.Biography == "" {
		bio = policy.Sanitize(bio)
		bio = strings.ReplaceAll(bio, "\n", " ")
		artist.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
	}
}

func (e *externalInfo) setExternalUrl(artist *model.Artist, url string) {
	if artist.ExternalUrl == "" {
		artist.ExternalUrl = url
	}
}

func (e *externalInfo) setMbzID(artist *model.Artist, mbID string) {
	if artist.MbzArtistID == "" {
		artist.MbzArtistID = mbID
	}
}

func (e *externalInfo) setSmallImageUrl(artist *model.Artist, url string) {
	if artist.SmallImageUrl == "" {
		artist.SmallImageUrl = url
	}
}

func (e *externalInfo) setMediumImageUrl(artist *model.Artist, url string) {
	if artist.MediumImageUrl == "" {
		artist.MediumImageUrl = url
	}
}

func (e *externalInfo) setLargeImageUrl(artist *model.Artist, url string) {
	if artist.LargeImageUrl == "" {
		artist.LargeImageUrl = url
	}
}

func (e *externalInfo) loadSimilar(ctx context.Context, artist *model.Artist, includeNotPresent bool) error {
	var ids []string
	for _, sa := range artist.SimilarArtists {
		if sa.ID == UnavailableArtistID {
			continue
		}
		ids = append(ids, sa.ID)
	}

	similar, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"id": ids},
	})
	if err != nil {
		return err
	}

	// Use a map and iterate through original array, to keep the same order
	artistMap := make(map[string]model.Artist)
	for _, sa := range similar {
		artistMap[sa.ID] = sa
	}

	var loaded model.Artists
	for _, sa := range artist.SimilarArtists {
		la, ok := artistMap[sa.ID]
		if !ok {
			if !includeNotPresent {
				continue
			}
			la = sa
			la.ID = UnavailableArtistID
		}
		loaded = append(loaded, la)
	}
	artist.SimilarArtists = loaded
	return nil
}
