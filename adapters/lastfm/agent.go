package lastfm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"golang.org/x/net/html"
)

const (
	lastFMAgentName    = "lastfm"
	sessionKeyProperty = "LastFMSessionKey"
)

var ignoredBiographies = []string{
	// Unknown Artist
	`<a href="https://www.last.fm/music/`,
}

type lastfmAgent struct {
	ds           model.DataStore
	sessionKeys  *agents.SessionKeys
	apiKey       string
	secret       string
	lang         string
	client       *client
	httpClient   httpDoer
	getInfoMutex sync.Mutex
}

func lastFMConstructor(ds model.DataStore) *lastfmAgent {
	if !conf.Server.LastFM.Enabled || conf.Server.LastFM.ApiKey == "" || conf.Server.LastFM.Secret == "" {
		return nil
	}
	l := &lastfmAgent{
		ds:          ds,
		lang:        conf.Server.LastFM.Language,
		apiKey:      conf.Server.LastFM.ApiKey,
		secret:      conf.Server.LastFM.Secret,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := cache.NewHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.httpClient = chc
	l.client = newClient(l.apiKey, l.secret, l.lang, chc)
	return l
}

func (l *lastfmAgent) AgentName() string {
	return lastFMAgentName
}

var imageRegex = regexp.MustCompile(`u\/(\d+)`)

func (l *lastfmAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	a, err := l.callAlbumGetInfo(ctx, name, artist, mbid)
	if err != nil {
		return nil, err
	}

	return &agents.AlbumInfo{
		Name:        a.Name,
		MBID:        a.MBID,
		Description: a.Description.Summary,
		URL:         a.URL,
	}, nil
}

func (l *lastfmAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	a, err := l.callAlbumGetInfo(ctx, name, artist, mbid)
	if err != nil {
		return nil, err
	}

	// Last.fm can return duplicate sizes.
	seenSizes := map[int]bool{}
	images := make([]agents.ExternalImage, 0)

	// This assumes that Last.fm returns images with size small, medium, and large.
	// This is true as of December 29, 2022
	for _, img := range a.Image {
		size := imageRegex.FindStringSubmatch(img.URL)
		// Last.fm can return images without URL
		if len(size) == 0 || len(size[0]) < 4 {
			log.Trace(ctx, "LastFM/albuminfo image URL does not match expected regex or is empty", "url", img.URL, "size", img.Size)
			continue
		}
		numericSize, err := strconv.Atoi(size[0][2:])
		if err != nil {
			log.Error(ctx, "LastFM/albuminfo image URL does not match expected regex", "url", img.URL, "size", img.Size, err)
			return nil, err
		}
		if _, exists := seenSizes[numericSize]; !exists {
			images = append(images, agents.ExternalImage{
				Size: numericSize,
				URL:  img.URL,
			})
			seenSizes[numericSize] = true
		}
	}
	return images, nil
}

func (l *lastfmAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name)
	if err != nil {
		return "", err
	}
	if a.MBID == "" {
		return "", agents.ErrNotFound
	}
	return a.MBID, nil
}

func (l *lastfmAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name)
	if err != nil {
		return "", err
	}
	if a.URL == "" {
		return "", agents.ErrNotFound
	}
	return a.URL, nil
}

func (l *lastfmAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name)
	if err != nil {
		return "", err
	}
	a.Bio.Summary = strings.TrimSpace(a.Bio.Summary)
	if a.Bio.Summary == "" {
		return "", agents.ErrNotFound
	}
	for _, ign := range ignoredBiographies {
		if strings.HasPrefix(a.Bio.Summary, ign) {
			return "", nil
		}
	}
	return a.Bio.Summary, nil
}

func (l *lastfmAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	resp, err := l.callArtistGetSimilar(ctx, name, limit)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}
	var res []agents.Artist
	for _, a := range resp {
		res = append(res, agents.Artist{
			Name: a.Name,
			MBID: a.MBID,
		})
	}
	return res, nil
}

func (l *lastfmAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	resp, err := l.callArtistGetTopTracks(ctx, artistName, count)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}
	var res []agents.Song
	for _, t := range resp {
		res = append(res, agents.Song{
			Name: t.Name,
			MBID: t.MBID,
		})
	}
	return res, nil
}

var (
	artistOpenGraphQuery = cascadia.MustCompile(`html > head > meta[property="og:image"]`)
	artistIgnoredImage   = "2a96cbd8b46e442fc41c2b86b821562f" // Last.fm artist placeholder image name
)

func (l *lastfmAgent) GetArtistImages(ctx context.Context, _, name, mbid string) ([]agents.ExternalImage, error) {
	log.Debug(ctx, "Getting artist images from Last.fm", "name", name)
	a, err := l.callArtistGetInfo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get artist info: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create artist image request: %w", err)
	}
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get artist url: %w", err)
	}
	defer resp.Body.Close()

	node, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var res []agents.ExternalImage
	n := cascadia.Query(node, artistOpenGraphQuery)
	if n == nil {
		return res, nil
	}
	for _, attr := range n.Attr {
		if attr.Key != "content" {
			continue
		}
		if strings.Contains(attr.Val, artistIgnoredImage) {
			log.Debug(ctx, "Artist image is ignored default image", "name", name, "url", attr.Val)
			return res, nil
		}

		res = []agents.ExternalImage{
			{URL: attr.Val},
		}
	}
	return res, nil
}

func (l *lastfmAgent) callAlbumGetInfo(ctx context.Context, name, artist, mbid string) (*Album, error) {
	a, err := l.client.albumGetInfo(ctx, name, artist, mbid)
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)

	if mbid != "" && (isLastFMError && lfErr.Code == 6) {
		log.Debug(ctx, "LastFM/album.getInfo could not find album by mbid, trying again", "album", name, "mbid", mbid)
		return l.callAlbumGetInfo(ctx, name, artist, "")
	}

	if err != nil {
		if isLastFMError && lfErr.Code == 6 {
			log.Debug(ctx, "Album not found", "album", name, "mbid", mbid, err)
		} else {
			log.Error(ctx, "Error calling LastFM/album.getInfo", "album", name, "mbid", mbid, err)
		}
		return nil, err
	}
	return a, nil
}

func (l *lastfmAgent) callArtistGetInfo(ctx context.Context, name string) (*Artist, error) {
	l.getInfoMutex.Lock()
	defer l.getInfoMutex.Unlock()

	a, err := l.client.artistGetInfo(ctx, name)
	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getInfo", "artist", name, err)
		return nil, err
	}
	return a, nil
}

func (l *lastfmAgent) callArtistGetSimilar(ctx context.Context, name string, limit int) ([]Artist, error) {
	s, err := l.client.artistGetSimilar(ctx, name, limit)
	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getSimilar", "artist", name, err)
		return nil, err
	}
	return s.Artists, nil
}

func (l *lastfmAgent) callArtistGetTopTracks(ctx context.Context, artistName string, count int) ([]Track, error) {
	t, err := l.client.artistGetTopTracks(ctx, artistName, count)
	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getTopTracks", "artist", artistName, err)
		return nil, err
	}
	return t.Track, nil
}

func (l *lastfmAgent) getArtistForScrobble(track *model.MediaFile, role model.Role, displayName string) string {
	if conf.Server.LastFM.ScrobbleFirstArtistOnly && len(track.Participants[role]) > 0 {
		return track.Participants[role][0].Name
	}
	return displayName
}

func (l *lastfmAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	err = l.client.updateNowPlaying(ctx, sk, ScrobbleInfo{
		artist:      l.getArtistForScrobble(track, model.RoleArtist, track.Artist),
		track:       track.Title,
		album:       track.Album,
		trackNumber: track.TrackNumber,
		mbid:        track.MbzRecordingID,
		duration:    int(track.Duration),
		albumArtist: l.getArtistForScrobble(track, model.RoleAlbumArtist, track.AlbumArtist),
	})
	if err != nil {
		log.Warn(ctx, "Last.fm client.updateNowPlaying returned error", "track", track.Title, err)
		return errors.Join(err, scrobbler.ErrUnrecoverable)
	}
	return nil
}

func (l *lastfmAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return errors.Join(err, scrobbler.ErrNotAuthorized)
	}

	if s.Duration <= 30 {
		log.Debug(ctx, "Skipping Last.fm scrobble for short song", "track", s.Title, "duration", s.Duration)
		return nil
	}
	err = l.client.scrobble(ctx, sk, ScrobbleInfo{
		artist:      l.getArtistForScrobble(&s.MediaFile, model.RoleArtist, s.Artist),
		track:       s.Title,
		album:       s.Album,
		trackNumber: s.TrackNumber,
		mbid:        s.MbzRecordingID,
		duration:    int(s.Duration),
		albumArtist: l.getArtistForScrobble(&s.MediaFile, model.RoleAlbumArtist, s.AlbumArtist),
		timestamp:   s.TimeStamp,
	})
	if err == nil {
		return nil
	}
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)
	if !isLastFMError {
		log.Warn(ctx, "Last.fm client.scrobble returned error", "track", s.Title, err)
		return errors.Join(err, scrobbler.ErrRetryLater)
	}
	if lfErr.Code == 11 || lfErr.Code == 16 {
		return errors.Join(err, scrobbler.ErrRetryLater)
	}
	return errors.Join(err, scrobbler.ErrUnrecoverable)
}

func (l *lastfmAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func init() {
	conf.AddHook(func() {
		agents.Register(lastFMAgentName, func(ds model.DataStore) agents.Interface {
			// This is a workaround for the fact that a (Interface)(nil) is not the same as a (*lastfmAgent)(nil)
			// See https://go.dev/doc/faq#nil_error
			a := lastFMConstructor(ds)
			if a != nil {
				return a
			}
			return nil
		})
		scrobbler.Register(lastFMAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
			// Same as above - this is a workaround for the fact that a (Scrobbler)(nil) is not the same as a (*lastfmAgent)(nil)
			// See https://go.dev/doc/faq#nil_error
			a := lastFMConstructor(ds)
			if a != nil {
				return a
			}
			return nil
		})
	})
}
