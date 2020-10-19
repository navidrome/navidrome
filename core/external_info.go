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
	ArtistInfo(ctx context.Context, artistId string, includeNotPresent bool, count int) (*model.ArtistInfo, error)
}

type LastFMClient interface {
	ArtistGetInfo(ctx context.Context, name string) (*lastfm.Artist, error)
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

func (e *externalInfo) ArtistInfo(ctx context.Context, artistId string,
	includeNotPresent bool, count int) (*model.ArtistInfo, error) {
	info := model.ArtistInfo{ID: artistId}

	artist, err := e.ds.Artist(ctx).Get(artistId)
	if err != nil {
		return nil, err
	}
	info.Name = artist.Name

	// TODO Load from local: artist.jpg/png/webp, artist.json (with the remaining info)

	var wg sync.WaitGroup
	e.callArtistInfo(ctx, artist, includeNotPresent, &wg, &info)
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

func (e *externalInfo) callArtistInfo(ctx context.Context, artist *model.Artist, includeNotPresent bool,
	wg *sync.WaitGroup, info *model.ArtistInfo) {
	if e.lfm != nil {
		log.Debug(ctx, "Calling Last.FM ArtistGetInfo", "artist", artist.Name)
		wg.Add(1)
		go func() {
			start := time.Now()
			defer wg.Done()
			lfmArtist, err := e.lfm.ArtistGetInfo(nil, artist.Name)
			if err != nil {
				log.Error(ctx, "Error calling Last.FM", "artist", artist.Name, err)
			} else {
				log.Debug(ctx, "Got info from Last.FM", "artist", artist.Name, "info", lfmArtist.Bio.Summary, "elapsed", time.Since(start))
			}
			e.setBio(info, lfmArtist.Bio.Summary)
			e.setLastFMUrl(info, lfmArtist.URL)
			e.setMbzID(info, lfmArtist.MBID)
			e.setSimilar(ctx, info, lfmArtist.Similar.Artists, includeNotPresent)
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
			spfImages, err := e.spf.ArtistImages(nil, artist.Name)
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
	if info.Bio == "" {
		bio = policy.Sanitize(bio)
		bio = strings.ReplaceAll(bio, "\n", " ")
		info.Bio = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
	}
}

func (e *externalInfo) setLastFMUrl(info *model.ArtistInfo, url string) {
	if info.LastFMUrl == "" {
		info.LastFMUrl = url
	}
}

func (e *externalInfo) setMbzID(info *model.ArtistInfo, mbzID string) {
	if info.MbzID == "" {
		info.MbzID = mbzID
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

func (e *externalInfo) setSimilar(ctx context.Context, info *model.ArtistInfo, artists []lastfm.Artist, includeNotPresent bool) {
	if len(info.Similar) == 0 {
		var notPresent []string

		// First select artists that are present.
		for _, s := range artists {
			sa, err := e.ds.Artist(ctx).FindByName(s.Name)
			if err != nil {
				notPresent = append(notPresent, s.Name)
				continue
			}
			info.Similar = append(info.Similar, *sa)
		}

		// Then fill up with non-present artists
		if includeNotPresent {
			for _, s := range notPresent {
				sa := model.Artist{ID: "-1", Name: s}
				info.Similar = append(info.Similar, sa)
			}
		}
	}
}
