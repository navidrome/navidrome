package subsonic

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/filter"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) getAlbumList(r *http.Request) (model.Albums, int64, error) {
	p := req.Params(r)
	typ, err := p.String("type")
	if err != nil {
		return nil, 0, err
	}

	var opts filter.Options
	switch typ {
	case "newest":
		opts = filter.AlbumsByNewest()
	case "recent":
		opts = filter.AlbumsByRecent()
	case "random":
		opts = filter.AlbumsByRandom()
	case "alphabeticalByName":
		opts = filter.AlbumsByName()
	case "alphabeticalByArtist":
		opts = filter.AlbumsByArtist()
	case "frequent":
		opts = filter.AlbumsByFrequent()
	case "starred":
		opts = filter.AlbumsByStarred()
	case "highest":
		opts = filter.AlbumsByRating()
	case "byGenre":
		genre, err := p.String("genre")
		if err != nil {
			return nil, 0, err
		}
		opts = filter.AlbumsByGenre(genre)
	case "byYear":
		fromYear, err := p.Int("fromYear")
		if err != nil {
			return nil, 0, err
		}
		toYear, err := p.Int("toYear")
		if err != nil {
			return nil, 0, err
		}
		opts = filter.AlbumsByYear(fromYear, toYear)
	default:
		log.Error(r, "albumList type not implemented", "type", typ)
		return nil, 0, newError(responses.ErrorGeneric, "type '%s' not implemented", typ)
	}

	opts.Offset = p.IntOr("offset", 0)
	opts.Max = min(p.IntOr("size", 10), 500)
	albums, err := api.ds.Album(r.Context()).GetAllWithoutGenres(opts)

	if err != nil {
		log.Error(r, "Error retrieving albums", err)
		return nil, 0, newError(responses.ErrorGeneric, "internal error")
	}

	count, err := api.ds.Album(r.Context()).CountAll(opts)
	if err != nil {
		log.Error(r, "Error counting albums", err)
		return nil, 0, newError(responses.ErrorGeneric, "internal error")
	}

	return albums, count, nil
}

func (api *Router) GetAlbumList(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, count, err := api.getAlbumList(r)
	if err != nil {
		return nil, err
	}

	w.Header().Set("x-total-count", strconv.Itoa(int(count)))

	response := newResponse()
	response.AlbumList = &responses.AlbumList{
		Album: slice.MapWithArg(albums, r.Context(), childFromAlbum),
	}
	return response, nil
}

func (api *Router) GetAlbumList2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, pageCount, err := api.getAlbumList(r)
	if err != nil {
		return nil, err
	}

	w.Header().Set("x-total-count", strconv.FormatInt(pageCount, 10))

	response := newResponse()
	response.AlbumList2 = &responses.AlbumList{
		Album: slice.MapWithArg(albums, r.Context(), childFromAlbum),
	}
	return response, nil
}

func (api *Router) GetStarred(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	options := filter.Starred()
	artists, err := api.ds.Artist(ctx).GetAll(options)
	if err != nil {
		log.Error(r, "Error retrieving starred artists", err)
		return nil, err
	}
	albums, err := api.ds.Album(ctx).GetAllWithoutGenres(options)
	if err != nil {
		log.Error(r, "Error retrieving starred albums", err)
		return nil, err
	}
	mediaFiles, err := api.ds.MediaFile(ctx).GetAll(options)
	if err != nil {
		log.Error(r, "Error retrieving starred mediaFiles", err)
		return nil, err
	}

	response := newResponse()
	response.Starred = &responses.Starred{}
	response.Starred.Artist = slice.MapWithArg(artists, r, toArtist)
	response.Starred.Album = slice.MapWithArg(albums, ctx, childFromAlbum)
	response.Starred.Song = slice.MapWithArg(mediaFiles, ctx, childFromMediaFile)
	return response, nil
}

func (api *Router) GetStarred2(r *http.Request) (*responses.Subsonic, error) {
	resp, err := api.GetStarred(r)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Starred2 = resp.Starred
	return response, nil
}

func (api *Router) GetNowPlaying(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	npInfo, err := api.scrobbler.GetNowPlaying(ctx)
	if err != nil {
		log.Error(r, "Error retrieving now playing list", err)
		return nil, err
	}

	response := newResponse()
	response.NowPlaying = &responses.NowPlaying{}
	var i int32
	response.NowPlaying.Entry = slice.Map(npInfo, func(np scrobbler.NowPlayingInfo) responses.NowPlayingEntry {
		return responses.NowPlayingEntry{
			Child:      childFromMediaFile(ctx, np.MediaFile),
			UserName:   np.Username,
			MinutesAgo: int32(time.Since(np.Start).Minutes()),
			PlayerId:   i + 1, // Fake numeric playerId, it does not seem to be used for anything
			PlayerName: np.PlayerName,
		}
	})
	return response, nil
}

func (api *Router) GetRandomSongs(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	size := min(p.IntOr("size", 10), 500)
	genre, _ := p.String("genre")
	fromYear := p.IntOr("fromYear", 0)
	toYear := p.IntOr("toYear", 0)

	songs, err := api.getSongs(r.Context(), 0, size, filter.SongsByRandom(genre, fromYear, toYear))
	if err != nil {
		log.Error(r, "Error retrieving random songs", err)
		return nil, err
	}

	response := newResponse()
	response.RandomSongs = &responses.Songs{}
	response.RandomSongs.Songs = slice.MapWithArg(songs, r.Context(), childFromMediaFile)
	return response, nil
}

func (api *Router) GetSongsByGenre(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	count := min(p.IntOr("count", 10), 500)
	offset := p.IntOr("offset", 0)
	genre, _ := p.String("genre")

	songs, err := api.getSongs(r.Context(), offset, count, filter.SongsByGenre(genre))
	if err != nil {
		log.Error(r, "Error retrieving random songs", err)
		return nil, err
	}

	response := newResponse()
	response.SongsByGenre = &responses.Songs{}
	response.SongsByGenre.Songs = slice.MapWithArg(songs, r.Context(), childFromMediaFile)
	return response, nil
}

func (api *Router) getSongs(ctx context.Context, offset, size int, opts filter.Options) (model.MediaFiles, error) {
	opts.Offset = offset
	opts.Max = size
	return api.ds.MediaFile(ctx).GetAll(opts)
}
