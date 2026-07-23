package subsonic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/server/imghttp"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/gravatar"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if !conf.Server.EnableGravatar {
		return api.getPlaceHolderAvatar(w, r)
	}
	p := req.Params(r)
	username, err := p.String("username")
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	u, err := api.ds.User(ctx).FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if u.Email == "" {
		log.Warn(ctx, "User needs an email for gravatar to work", "username", username)
		return api.getPlaceHolderAvatar(w, r)
	}
	http.Redirect(w, r, gravatar.Url(u.Email, 0), http.StatusFound) //nolint:gosec // URL is not constructed from user input
	return nil, nil
}

func (api *Router) getPlaceHolderAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	f, err := resources.FS().Open(consts.PlaceholderAvatar)
	if err != nil {
		log.Error(r, "Image not found", err)
		return nil, newError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	_, _ = io.Copy(w, f)

	return nil, nil
}

func (api *Router) GetCoverArt(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	// If context is already canceled, discard request without further processing
	if r.Context().Err() != nil {
		return nil, nil //nolint:nilerr
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	p := req.Params(r)
	id, _ := p.String("id")
	size := p.IntOr("size", 0)
	square := p.BoolOr("square", false)

	img, err := api.artwork.GetOrPlaceholder(ctx, id, size, square)
	switch {
	case errors.Is(err, context.Canceled):
		return nil, nil
	case errors.Is(err, model.ErrNotFound):
		log.Warn(r, "Couldn't find coverArt", "id", id, err)
		return nil, newError(responses.ErrorDataNotFound, "Artwork not found")
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		return nil, err
	}

	// Access control: the serving path reads persisted state by id, bypassing the library and
	// private-playlist filters. On this authenticated path, fall back to the placeholder for an
	// entity the caller cannot see, so a guessed id can't leak artwork (matches the legacy load).
	if id != "" && !img.Placeholder && !api.artworkAccessible(ctx, id) {
		_ = img.Close()
		img = artwork.PlaceholderFor(id)
	}
	defer img.Close()

	artID, _ := model.ParseArtworkID(id)
	if imghttp.WriteImageHeaders(w, r, img, artID.Hash) {
		return nil, nil
	}
	cnt, err := io.Copy(w, img)
	if err != nil {
		log.Warn(ctx, "Error sending image", "count", cnt, err)
	}

	return nil, err
}

// artworkAccessible reports whether the caller may view the artwork for id, by resolving the
// underlying entity through the request-scoped (filtered) repositories. Radios are global and
// always accessible; an unparsable/unknown id defers to GetEntityByID.
func (api *Router) artworkAccessible(ctx context.Context, id string) bool {
	artID, err := model.ParseArtworkID(id)
	if err != nil {
		_, err := model.GetEntityByID(ctx, api.ds, id)
		return err == nil
	}
	var lookupErr error
	switch artID.Kind {
	case model.KindArtistArtwork:
		_, lookupErr = api.ds.Artist(ctx).Get(artID.ID)
	case model.KindAlbumArtwork:
		_, lookupErr = api.ds.Album(ctx).Get(artID.ID)
	case model.KindMediaFileArtwork:
		_, lookupErr = api.ds.MediaFile(ctx).Get(artID.ID)
	case model.KindPlaylistArtwork:
		_, lookupErr = api.ds.Playlist(ctx).Get(artID.ID)
	case model.KindDiscArtwork:
		albumID, _, perr := model.ParseDiscArtworkID(artID.ID)
		if perr != nil {
			return false
		}
		_, lookupErr = api.ds.Album(ctx).Get(albumID)
	default: // radio and anything else has no per-user artwork access control
		return true
	}
	return lookupErr == nil
}

func (api *Router) GetLyrics(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	artist, _ := p.String("artist")
	title, _ := p.String("title")
	response := newResponse()
	lyricsResponse := responses.Lyrics{}
	response.Lyrics = &lyricsResponse
	structuredLyrics, err := api.lyrics.GetLyricsByArtistTitle(r.Context(), artist, title)
	if err != nil {
		return nil, err
	}

	mainLyric, ok := structuredLyrics.Main()
	if !ok {
		return response, nil
	}

	lyricsResponse.Artist = artist
	lyricsResponse.Title = title

	var lyricsText strings.Builder
	for _, line := range mainLyric.Line {
		lyricsText.WriteString(line.Value + "\n")
	}
	lyricsResponse.Value = lyricsText.String()

	return response, nil
}

func (api *Router) GetLyricsBySongId(r *http.Request) (*responses.Subsonic, error) {
	id, err := req.Params(r).String("id")
	if err != nil {
		return nil, err
	}

	mediaFile, err := api.ds.MediaFile(r.Context()).Get(id)
	if err != nil {
		return nil, err
	}

	structuredLyrics, err := api.lyrics.GetLyrics(r.Context(), mediaFile)
	if err != nil {
		return nil, err
	}

	enhanced, _ := req.Params(r).Bool("enhanced")

	response := newResponse()
	response.LyricsList = buildLyricsList(mediaFile, structuredLyrics, enhanced)

	return response, nil
}
