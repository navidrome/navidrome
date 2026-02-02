package tidal

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/slice"
)

const tidalAgentName = "tidal"
const tidalArtistSearchLimit = 20
const tidalAlbumSearchLimit = 10
const tidalTrackSearchLimit = 10
const tidalArtistURLBase = "https://tidal.com/browse/artist/"

type tidalAgent struct {
	ds     model.DataStore
	client *client
}

func tidalConstructor(ds model.DataStore) agents.Interface {
	if conf.Server.Tidal.ClientID == "" || conf.Server.Tidal.ClientSecret == "" {
		return nil
	}
	l := &tidalAgent{
		ds: ds,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := cache.NewHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(conf.Server.Tidal.ClientID, conf.Server.Tidal.ClientSecret, chc)
	return l
}

func (t *tidalAgent) AgentName() string {
	return tidalAgentName
}

func (t *tidalAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			log.Warn(ctx, "Artist not found in Tidal", "artist", name)
		} else {
			log.Error(ctx, "Error calling Tidal", "artist", name, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	for _, img := range artist.Attributes.Picture {
		res = append(res, agents.ExternalImage{
			URL:  img.URL,
			Size: img.Width,
		})
	}

	// Sort images by size descending
	if len(res) == 0 {
		return nil, agents.ErrNotFound
	}

	return res, nil
}

func (t *tidalAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		return nil, err
	}

	similar, err := t.client.getSimilarArtists(ctx, artist.ID, limit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	res := slice.Map(similar, func(a ArtistResource) agents.Artist {
		return agents.Artist{
			Name: a.Attributes.Name,
		}
	})

	return res, nil
}

func (t *tidalAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	artist, err := t.searchArtist(ctx, artistName)
	if err != nil {
		return nil, err
	}

	tracks, err := t.client.getArtistTopTracks(ctx, artist.ID, count)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	res := slice.Map(tracks, func(track TrackResource) agents.Song {
		return agents.Song{
			Name:     track.Attributes.Title,
			ISRC:     track.Attributes.ISRC,
			Duration: uint32(track.Attributes.Duration * 1000), // Convert seconds to milliseconds
		}
	})

	return res, nil
}

func (t *tidalAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		return "", err
	}

	return tidalArtistURLBase + artist.ID, nil
}

func (t *tidalAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		return "", err
	}

	bio, err := t.client.getArtistBio(ctx, artist.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", agents.ErrNotFound
		}
		log.Error(ctx, "Error getting artist bio from Tidal", "artist", name, err)
		return "", err
	}

	return bio, nil
}

func (t *tidalAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	album, err := t.searchAlbum(ctx, name, artist)
	if err != nil {
		return nil, err
	}

	// Try to get album review/description
	description, err := t.client.getAlbumReview(ctx, album.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Warn(ctx, "Error getting album review from Tidal", "album", name, err)
	}

	return &agents.AlbumInfo{
		Name:        album.Attributes.Title,
		Description: description,
		URL:         "https://tidal.com/browse/album/" + album.ID,
	}, nil
}

func (t *tidalAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	album, err := t.searchAlbum(ctx, name, artist)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			log.Warn(ctx, "Album not found in Tidal", "album", name, "artist", artist)
		} else {
			log.Error(ctx, "Error calling Tidal for album", "album", name, "artist", artist, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	for _, img := range album.Attributes.Cover {
		res = append(res, agents.ExternalImage{
			URL:  img.URL,
			Size: img.Width,
		})
	}

	if len(res) == 0 {
		return nil, agents.ErrNotFound
	}

	return res, nil
}

func (t *tidalAgent) GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]agents.Song, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get similar artists
	similarArtists, err := t.client.getSimilarArtists(ctx, artist.ID, 5)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	if len(similarArtists) == 0 {
		return nil, agents.ErrNotFound
	}

	// Get top tracks from similar artists
	var songs []agents.Song
	tracksPerArtist := (count / len(similarArtists)) + 1

	for _, simArtist := range similarArtists {
		tracks, err := t.client.getArtistTopTracks(ctx, simArtist.ID, tracksPerArtist)
		if err != nil {
			log.Warn(ctx, "Failed to get top tracks for similar artist", "artist", simArtist.Attributes.Name, err)
			continue
		}

		for _, track := range tracks {
			songs = append(songs, agents.Song{
				Name:     track.Attributes.Title,
				Artist:   simArtist.Attributes.Name,
				ISRC:     track.Attributes.ISRC,
				Duration: uint32(track.Attributes.Duration * 1000),
			})
			if len(songs) >= count {
				return songs, nil
			}
		}
	}

	if len(songs) == 0 {
		return nil, agents.ErrNotFound
	}

	return songs, nil
}

func (t *tidalAgent) GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]agents.Song, error) {
	track, err := t.searchTrack(ctx, name, artist)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			log.Warn(ctx, "Track not found in Tidal", "track", name, "artist", artist)
		} else {
			log.Error(ctx, "Error searching track in Tidal", "track", name, "artist", artist, err)
		}
		return nil, err
	}

	// Get track radio (similar tracks)
	similarTracks, err := t.client.getTrackRadio(ctx, track.ID, count)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		log.Error(ctx, "Error getting track radio from Tidal", "trackId", track.ID, err)
		return nil, err
	}

	if len(similarTracks) == 0 {
		return nil, agents.ErrNotFound
	}

	res := slice.Map(similarTracks, func(track TrackResource) agents.Song {
		return agents.Song{
			Name:     track.Attributes.Title,
			ISRC:     track.Attributes.ISRC,
			Duration: uint32(track.Attributes.Duration * 1000),
		}
	})

	return res, nil
}

func (t *tidalAgent) searchTrack(ctx context.Context, trackName, artistName string) (*TrackResource, error) {
	tracks, err := t.client.searchTracks(ctx, trackName, artistName, tidalTrackSearchLimit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	if len(tracks) == 0 {
		return nil, agents.ErrNotFound
	}

	// Find exact match (case-insensitive)
	for i := range tracks {
		if strings.EqualFold(tracks[i].Attributes.Title, trackName) {
			log.Trace(ctx, "Found track in Tidal", "title", tracks[i].Attributes.Title, "id", tracks[i].ID)
			return &tracks[i], nil
		}
	}

	// If no exact match, check if first result is close enough
	log.Trace(ctx, "No exact track match in Tidal", "searched", trackName, "found", tracks[0].Attributes.Title)
	return nil, agents.ErrNotFound
}

func (t *tidalAgent) searchArtist(ctx context.Context, name string) (*ArtistResource, error) {
	artists, err := t.client.searchArtists(ctx, name, tidalArtistSearchLimit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	if len(artists) == 0 {
		return nil, agents.ErrNotFound
	}

	// Find exact match (case-insensitive)
	for i := range artists {
		if strings.EqualFold(artists[i].Attributes.Name, name) {
			log.Trace(ctx, "Found artist in Tidal", "name", artists[i].Attributes.Name, "id", artists[i].ID)
			return &artists[i], nil
		}
	}

	// If no exact match, check if first result is close enough
	log.Trace(ctx, "No exact artist match in Tidal", "searched", name, "found", artists[0].Attributes.Name)
	return nil, agents.ErrNotFound
}

func (t *tidalAgent) searchAlbum(ctx context.Context, albumName, artistName string) (*AlbumResource, error) {
	albums, err := t.client.searchAlbums(ctx, albumName, artistName, tidalAlbumSearchLimit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	if len(albums) == 0 {
		return nil, agents.ErrNotFound
	}

	// Find exact match (case-insensitive)
	for i := range albums {
		if strings.EqualFold(albums[i].Attributes.Title, albumName) {
			log.Trace(ctx, "Found album in Tidal", "title", albums[i].Attributes.Title, "id", albums[i].ID)
			return &albums[i], nil
		}
	}

	// If no exact match, check if first result is close enough
	log.Trace(ctx, "No exact album match in Tidal", "searched", albumName, "found", albums[0].Attributes.Title)
	return nil, agents.ErrNotFound
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Tidal.Enabled {
			agents.Register(tidalAgentName, tidalConstructor)
		}
	})
}
