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
	"github.com/navidrome/navidrome/utils"
)

type AlbumListController struct {
	ds        model.DataStore
	scrobbler scrobbler.PlayTracker
}

func NewAlbumListController(ds model.DataStore, scrobbler scrobbler.PlayTracker) *AlbumListController {
	c := &AlbumListController{
		ds:        ds,
		scrobbler: scrobbler,
	}
	return c
}

func (c *AlbumListController) getAlbumList(r *http.Request) (model.Albums, int64, error) {
	typ, err := requiredParamString(r, "type")
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
		genre, err := requiredParamString(r, "genre")
		if err != nil {
			return nil, 0, err
		}
		opts = filter.AlbumsByGenre(genre)
	case "byYear":
		fromYear, err := requiredParamInt(r, "fromYear")
		if err != nil {
			return nil, 0, err
		}
		toYear, err := requiredParamInt(r, "toYear")
		if err != nil {
			return nil, 0, err
		}
		opts = filter.AlbumsByYear(fromYear, toYear)
	default:
		log.Error(r, "albumList type not implemented", "type", typ)
		return nil, 0, newError(responses.ErrorGeneric, "type '%s' not implemented", typ)
	}

	opts.Offset = utils.ParamInt(r, "offset", 0)
	opts.Max = utils.MinInt(utils.ParamInt(r, "size", 10), 500)
	albums, err := c.ds.Album(r.Context()).GetAllWithoutGenres(opts)

	if err != nil {
		log.Error(r, "Error retrieving albums", "error", err)
		return nil, 0, newError(responses.ErrorGeneric, "internal error")
	}

	count, err := c.ds.Album(r.Context()).CountAll(opts)
	if err != nil {
		log.Error(r, "Error counting albums", "error", err)
		return nil, 0, newError(responses.ErrorGeneric, "internal error")
	}

	return albums, count, nil
}

func (c *AlbumListController) GetAlbumList(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, count, err := c.getAlbumList(r)
	if err != nil {
		return nil, err
	}

	w.Header().Set("x-total-count", strconv.Itoa(int(count)))

	response := newResponse()
	response.AlbumList = &responses.AlbumList{Album: childrenFromAlbums(r.Context(), albums)}
	return response, nil
}

func (c *AlbumListController) GetAlbumList2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, pageCount, err := c.getAlbumList(r)
	if err != nil {
		return nil, err
	}

	w.Header().Set("x-total-count", strconv.FormatInt(pageCount, 10))

	response := newResponse()
	response.AlbumList2 = &responses.AlbumList{Album: childrenFromAlbums(r.Context(), albums)}
	return response, nil
}

func (c *AlbumListController) GetStarred(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	options := filter.Starred()
	artists, err := c.ds.Artist(ctx).GetAll(options)
	if err != nil {
		log.Error(r, "Error retrieving starred artists", "error", err)
		return nil, err
	}
	albums, err := c.ds.Album(ctx).GetAllWithoutGenres(options)
	if err != nil {
		log.Error(r, "Error retrieving starred albums", "error", err)
		return nil, err
	}
	mediaFiles, err := c.ds.MediaFile(ctx).GetAll(options)
	if err != nil {
		log.Error(r, "Error retrieving starred mediaFiles", "error", err)
		return nil, err
	}

	response := newResponse()
	response.Starred = &responses.Starred{}
	response.Starred.Artist = toArtists(ctx, artists, getBaseArtistImageUrl(r))
	response.Starred.Album = childrenFromAlbums(r.Context(), albums)
	response.Starred.Song = childrenFromMediaFiles(r.Context(), mediaFiles)
	return response, nil
}

func (c *AlbumListController) GetStarred2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	resp, err := c.GetStarred(w, r)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Starred2 = resp.Starred
	return response, nil
}

func (c *AlbumListController) GetNowPlaying(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	npInfo, err := c.scrobbler.GetNowPlaying(ctx)
	if err != nil {
		log.Error(r, "Error retrieving now playing list", "error", err)
		return nil, err
	}

	response := newResponse()
	response.NowPlaying = &responses.NowPlaying{}
	response.NowPlaying.Entry = make([]responses.NowPlayingEntry, len(npInfo))
	for i, np := range npInfo {
		mf, err := c.ds.MediaFile(ctx).Get(np.TrackID)
		if err != nil {
			return nil, err
		}

		response.NowPlaying.Entry[i].Child = childFromMediaFile(ctx, *mf)
		response.NowPlaying.Entry[i].UserName = np.Username
		response.NowPlaying.Entry[i].MinutesAgo = int(time.Since(np.Start).Minutes())
		response.NowPlaying.Entry[i].PlayerId = i + 1 // Fake numeric playerId, it does not seem to be used for anything
		response.NowPlaying.Entry[i].PlayerName = np.PlayerName
	}
	return response, nil
}

func (c *AlbumListController) GetRandomSongs(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	size := utils.MinInt(utils.ParamInt(r, "size", 10), 500)
	genre := utils.ParamString(r, "genre")
	fromYear := utils.ParamInt(r, "fromYear", 0)
	toYear := utils.ParamInt(r, "toYear", 0)

	songs, err := c.getSongs(r.Context(), 0, size, filter.SongsByRandom(genre, fromYear, toYear))
	if err != nil {
		log.Error(r, "Error retrieving random songs", "error", err)
		return nil, err
	}

	response := newResponse()
	response.RandomSongs = &responses.Songs{}
	response.RandomSongs.Songs = childrenFromMediaFiles(r.Context(), songs)
	return response, nil
}

func (c *AlbumListController) GetSongsByGenre(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	count := utils.MinInt(utils.ParamInt(r, "count", 10), 500)
	offset := utils.MinInt(utils.ParamInt(r, "offset", 0), 500)
	genre := utils.ParamString(r, "genre")

	songs, err := c.getSongs(r.Context(), offset, count, filter.SongsByGenre(genre))
	if err != nil {
		log.Error(r, "Error retrieving random songs", "error", err)
		return nil, err
	}

	response := newResponse()
	response.SongsByGenre = &responses.Songs{}
	response.SongsByGenre.Songs = childrenFromMediaFiles(r.Context(), songs)
	return response, nil
}

func (c *AlbumListController) getSongs(ctx context.Context, offset, size int, opts filter.Options) (model.MediaFiles, error) {
	opts.Offset = offset
	opts.Max = size
	return c.ds.MediaFile(ctx).GetAll(opts)
}
