package core

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deluan/navidrome/core/lastfm"
	"github.com/deluan/navidrome/core/spotify"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/microcosm-cc/bluemonday"
)

const placeholderArtistImageSmallUrl = "https://lastfm.freetls.fastly.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageMediumUrl = "https://lastfm.freetls.fastly.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageLargeUrl = "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png"

type ExternalInfo interface {
	ArtistInfo(ctx context.Context, id string) (*model.ArtistInfo, error)
	SimilarArtists(ctx context.Context, id string, includeNotPresent bool, count int) (model.Artists, error)
	SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error)
}

type LastFMClient interface {
	ArtistGetInfo(ctx context.Context, name string) (*lastfm.Artist, error)
	ArtistGetSimilar(ctx context.Context, name string, limit int) ([]lastfm.Artist, error)
}

type SpotifyClient interface {
	ArtistImages(ctx context.Context, name string) ([]spotify.Image, error)
}

func NewExternalInfo(ds model.DataStore, lfm LastFMClient, spf SpotifyClient) ExternalInfo {
	return &externalInfo{ds: ds, lfm: lfm, spf: spf}
}

type externalInfo struct {
	ds  model.DataStore
	lfm LastFMClient
	spf SpotifyClient
}

func (e *externalInfo) getArtist(ctx context.Context, id string) (artist *model.Artist, err error) {
	var entity interface{}
	entity, err = GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return nil, err
	}

	switch v := entity.(type) {
	case *model.Artist:
		artist = v
	case *model.MediaFile:
		artist = &model.Artist{
			ID:   v.ArtistID,
			Name: v.Artist,
		}
	case *model.Album:
		artist = &model.Artist{
			ID:   v.AlbumArtistID,
			Name: v.Artist,
		}
	default:
		err = model.ErrNotFound
	}
	return
}

func (e *externalInfo) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	// TODO
	// Get Similar Artists
	// Get `count` songs from all similar artists, sorted randomly
	return nil, nil
}

func (e *externalInfo) SimilarArtists(ctx context.Context, id string, includeNotPresent bool, count int) (model.Artists, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	var result model.Artists
	var notPresent []string

	similar, err := e.lfm.ArtistGetSimilar(ctx, artist.Name, count)
	if err != nil {
		return nil, err
	}

	// First select artists that are present.
	for _, s := range similar {
		sa, err := e.ds.Artist(ctx).FindByName(s.Name)
		if err != nil {
			notPresent = append(notPresent, s.Name)
			continue
		}
		result = append(result, *sa)
	}

	// Then fill up with non-present artists
	if includeNotPresent {
		for _, s := range notPresent {
			sa := model.Artist{ID: "-1", Name: s}
			result = append(result, sa)
		}
	}

	return result, nil
}

func (e *externalInfo) ArtistInfo(ctx context.Context, id string) (*model.ArtistInfo, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	info := model.ArtistInfo{ID: artist.ID, Name: artist.Name}

	// TODO Load from local: artist.jpg/png/webp, artist.json (with the remaining info)

	var wg sync.WaitGroup
	e.callArtistInfo(ctx, artist, &wg, &info)
	e.callArtistImages(ctx, artist, &wg, &info)
	wg.Wait()

	// Use placeholders if could not get from external sources
	e.setBio(&info, "Biography not available")
	e.setSmallImageUrl(&info, placeholderArtistImageSmallUrl)
	e.setMediumImageUrl(&info, placeholderArtistImageMediumUrl)
	e.setLargeImageUrl(&info, placeholderArtistImageLargeUrl)

	log.Trace(ctx, "ArtistInfo collected", "artist", artist.Name, "info", info)

	return &info, nil
}

func (e *externalInfo) callArtistInfo(ctx context.Context, artist *model.Artist,
	wg *sync.WaitGroup, info *model.ArtistInfo) {
	if e.lfm != nil {
		log.Debug(ctx, "Calling Last.FM ArtistGetInfo", "artist", artist.Name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()
			lfmArtist, err := e.lfm.ArtistGetInfo(ctx, artist.Name)
			if err != nil {
				log.Error(ctx, "Error calling Last.FM", "artist", artist.Name, err)
			} else {
				log.Debug(ctx, "Got info from Last.FM", "artist", artist.Name, "info", lfmArtist.Bio.Summary, "elapsed", time.Since(start))
			}
			e.setBio(info, lfmArtist.Bio.Summary)
			e.setLastFMUrl(info, lfmArtist.URL)
			e.setMbzID(info, lfmArtist.MBID)
		}()
	}
}

func (e *externalInfo) callArtistImages(ctx context.Context, artist *model.Artist, wg *sync.WaitGroup, info *model.ArtistInfo) {
	if e.spf != nil {
		log.Debug(ctx, "Calling Spotify ArtistImages", "artist", artist.Name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()
			spfImages, err := e.spf.ArtistImages(ctx, artist.Name)
			if err != nil {
				log.Error(ctx, "Error calling Spotify", "artist", artist.Name, err)
			} else {
				log.Debug(ctx, "Got images from Spotify", "artist", artist.Name, "images", spfImages, "elapsed", time.Since(start))
			}

			sort.Slice(spfImages, func(i, j int) bool { return spfImages[i].Width > spfImages[j].Width })

			if len(spfImages) >= 1 {
				e.setLargeImageUrl(info, spfImages[0].URL)
			}
			if len(spfImages) >= 2 {
				e.setMediumImageUrl(info, spfImages[1].URL)
			}
			if len(spfImages) >= 3 {
				e.setSmallImageUrl(info, spfImages[2].URL)
			}
		}()
	}
}

func (e *externalInfo) setBio(info *model.ArtistInfo, bio string) {
	policy := bluemonday.UGCPolicy()
	if info.Biography == "" {
		bio = policy.Sanitize(bio)
		bio = strings.ReplaceAll(bio, "\n", " ")
		info.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
	}
}

func (e *externalInfo) setLastFMUrl(info *model.ArtistInfo, url string) {
	if info.LastFMUrl == "" {
		info.LastFMUrl = url
	}
}

func (e *externalInfo) setMbzID(info *model.ArtistInfo, mbID string) {
	if info.MBID == "" {
		info.MBID = mbID
	}
}

func (e *externalInfo) setSmallImageUrl(info *model.ArtistInfo, url string) {
	if info.SmallImageUrl == "" {
		info.SmallImageUrl = url
	}
}

func (e *externalInfo) setMediumImageUrl(info *model.ArtistInfo, url string) {
	if info.MediumImageUrl == "" {
		info.MediumImageUrl = url
	}
}

func (e *externalInfo) setLargeImageUrl(info *model.ArtistInfo, url string) {
	if info.LargeImageUrl == "" {
		info.LargeImageUrl = url
	}
}
