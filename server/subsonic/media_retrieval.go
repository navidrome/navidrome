package subsonic

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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
	u, err := api.ds.User().FindByUsername(ctx, username)
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

func (api *Router) GetLyrics(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	artist, _ := p.String("artist")
	title, _ := p.String("title")
	response := newResponse()
	lyricsResponse := responses.Lyrics{}
	response.Lyrics = &lyricsResponse
	structuredLyrics, err := api.lyrics.GetLyricsByArtistTitle(r.Context(), artist, title)
	if err != nil {
		log.Error(r.Context(), "Error getting lyrics", "artist", artist, "title", title, err)
		return nil, lyricsClientError(err)
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

	mediaFile, err := api.ds.MediaFile().Get(r.Context(), id)
	if err != nil {
		return nil, err
	}

	enhanced, _ := req.Params(r).Bool("enhanced")
	allSources, _ := req.Params(r).Bool("allSources")
	var structuredLyrics model.LyricList
	if enhanced {
		structuredLyrics, err = api.lyrics.GetLyricsVariants(r.Context(), mediaFile, allSources)
	} else {
		structuredLyrics, err = api.lyrics.GetLyrics(r.Context(), mediaFile)
	}
	if err != nil && (!enhanced || len(structuredLyrics) == 0) {
		log.Error(r.Context(), "Error getting lyrics", "id", id, err)
		return nil, lyricsClientError(err)
	}

	response := newResponse()
	response.LyricsList = buildLyricsList(mediaFile, structuredLyrics, enhanced)
	response.LyricsList.SourcesIncomplete = enhanced && err != nil
	if enhanced {
		response.LyricsList.RetryAfterSeconds = lyricsRetryAfterSeconds(err)
	}

	return response, nil
}

// GetLyricsVariants lists the addressable playback representations for one
// track. It performs no fetch and never returns document bytes.
func (api *Router) GetLyricsVariants(r *http.Request) (*responses.Subsonic, error) {
	id, err := req.Params(r).String("id")
	if err != nil {
		return nil, err
	}
	mediaFile, err := api.ds.MediaFile().Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	variants, err := api.lyrics.ListLyricsVariants(r.Context(), mediaFile)
	if err != nil {
		log.Error(r.Context(), "Error listing lyrics variants", "id", id, err)
	}
	response := newResponse()
	response.LyricsVariants = buildLyricsVariants(variants)
	return response, nil
}

// GetLyricsOriginal retrieves one exact source-native document for an opaque
// selection token. Login and the existing media-file library authorization both
// apply before the plugin is asked for bytes.
func (api *Router) GetLyricsOriginal(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	params := req.Params(r)
	id, err := params.String("id")
	if err != nil {
		return nil, err
	}
	selection, err := params.String("selection")
	if err != nil {
		return nil, err
	}
	plugin, identity, ok := model.DecodeLyricsSelection(selection)
	if !ok {
		return nil, newError(responses.ErrorGeneric, "invalid lyrics selection")
	}
	mediaFile, err := api.ds.MediaFile().Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	original, err := api.lyrics.GetLyricsOriginal(r.Context(), mediaFile, plugin, identity)
	if err != nil {
		if errors.Is(err, model.ErrLyricsOriginalUnavailable) {
			return nil, newError(responses.ErrorDataNotFound, "lyrics original unavailable")
		}
		return nil, err
	}
	w.Header().Set("Content-Type", original.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": lyricsOriginalFilename(mediaFile, original.Extension),
	}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(original.Document); err != nil {
		return nil, err
	}
	return nil, nil
}

func lyricsOriginalFilename(mf *model.MediaFile, extension string) string {
	base := strings.TrimSpace(mf.Title)
	if artist := strings.TrimSpace(mf.Artist); artist != "" && base != "" {
		base = artist + " - " + base
	}
	if base == "" {
		base = "lyrics"
	}
	base = strings.Map(func(char rune) rune {
		if char < 32 || strings.ContainsRune(`/\:*?"<>|`, char) {
			return '_'
		}
		return char
	}, base)
	if runes := []rune(base); len(runes) > 120 {
		base = string(runes[:120])
	}
	return base + extension
}
