package subsonic

import (
	"errors"
	"net/http"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type AlbumListController struct {
	listGen engine.ListGenerator
}

func NewAlbumListController(listGen engine.ListGenerator) *AlbumListController {
	c := &AlbumListController{
		listGen: listGen,
	}
	return c
}

func (c *AlbumListController) getNewAlbumList(r *http.Request) (engine.Entries, error) {
	typ, err := RequiredParamString(r, "type", "Required string parameter 'type' is not present")
	if err != nil {
		return nil, err
	}

	var filter engine.ListFilter
	switch typ {
	case "newest":
		filter = engine.ByNewest()
	case "recent":
		filter = engine.ByRecent()
	case "random":
		filter = engine.ByRandom()
	case "alphabeticalByName":
		filter = engine.ByName()
	case "alphabeticalByArtist":
		filter = engine.ByArtist()
	case "frequent":
		filter = engine.ByFrequent()
	case "starred":
		filter = engine.ByStarred()
	case "highest":
		filter = engine.ByRating()
	case "byGenre":
		filter = engine.ByGenre(utils.ParamString(r, "genre"))
	case "byYear":
		filter = engine.ByYear(utils.ParamInt(r, "fromYear", 0), utils.ParamInt(r, "toYear", 0))
	default:
		log.Error(r, "albumList type not implemented", "type", typ)
		return nil, errors.New("Not implemented!")
	}

	offset := utils.ParamInt(r, "offset", 0)
	size := utils.MinInt(utils.ParamInt(r, "size", 10), 500)

	albums, err := c.listGen.GetAlbums(r.Context(), offset, size, filter)
	if err != nil {
		log.Error(r, "Error retrieving albums", "error", err)
		return nil, errors.New("Internal Error")
	}

	return albums, nil
}

func (c *AlbumListController) GetAlbumList(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, err := c.getNewAlbumList(r)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, err.Error())
	}

	response := NewResponse()
	response.AlbumList = &responses.AlbumList{Album: ToChildren(r.Context(), albums)}
	return response, nil
}

func (c *AlbumListController) GetAlbumList2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	albums, err := c.getNewAlbumList(r)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, err.Error())
	}

	response := NewResponse()
	response.AlbumList2 = &responses.AlbumList{Album: ToAlbums(r.Context(), albums)}
	return response, nil
}

func (c *AlbumListController) GetStarred(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	artists, albums, mediaFiles, err := c.listGen.GetAllStarred(r.Context())
	if err != nil {
		log.Error(r, "Error retrieving starred media", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Starred = &responses.Starred{}
	response.Starred.Artist = ToArtists(artists)
	response.Starred.Album = ToChildren(r.Context(), albums)
	response.Starred.Song = ToChildren(r.Context(), mediaFiles)
	return response, nil
}

func (c *AlbumListController) GetStarred2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	artists, albums, mediaFiles, err := c.listGen.GetAllStarred(r.Context())
	if err != nil {
		log.Error(r, "Error retrieving starred media", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Starred2 = &responses.Starred{}
	response.Starred2.Artist = ToArtists(artists)
	response.Starred2.Album = ToAlbums(r.Context(), albums)
	response.Starred2.Song = ToChildren(r.Context(), mediaFiles)
	return response, nil
}

func (c *AlbumListController) GetNowPlaying(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	npInfos, err := c.listGen.GetNowPlaying(r.Context())
	if err != nil {
		log.Error(r, "Error retrieving now playing list", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.NowPlaying = &responses.NowPlaying{}
	response.NowPlaying.Entry = make([]responses.NowPlayingEntry, len(npInfos))
	for i, entry := range npInfos {
		response.NowPlaying.Entry[i].Child = ToChild(r.Context(), entry)
		response.NowPlaying.Entry[i].UserName = entry.UserName
		response.NowPlaying.Entry[i].MinutesAgo = entry.MinutesAgo
		response.NowPlaying.Entry[i].PlayerId = entry.PlayerId
		response.NowPlaying.Entry[i].PlayerName = entry.PlayerName
	}
	return response, nil
}

func (c *AlbumListController) GetRandomSongs(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	size := utils.MinInt(utils.ParamInt(r, "size", 10), 500)
	genre := utils.ParamString(r, "genre")
	fromYear := utils.ParamInt(r, "fromYear", 0)
	toYear := utils.ParamInt(r, "toYear", 0)

	songs, err := c.listGen.GetSongs(r.Context(), 0, size, engine.SongsByRandom(genre, fromYear, toYear))
	if err != nil {
		log.Error(r, "Error retrieving random songs", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.RandomSongs = &responses.Songs{}
	response.RandomSongs.Songs = ToChildren(r.Context(), songs)
	return response, nil
}

func (c *AlbumListController) GetSongsByGenre(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	count := utils.MinInt(utils.ParamInt(r, "count", 10), 500)
	offset := utils.MinInt(utils.ParamInt(r, "offset", 0), 500)
	genre := utils.ParamString(r, "genre")

	songs, err := c.listGen.GetSongs(r.Context(), offset, count, engine.SongsByGenre(genre))
	if err != nil {
		log.Error(r, "Error retrieving random songs", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.SongsByGenre = &responses.Songs{}
	response.SongsByGenre.Songs = ToChildren(r.Context(), songs)
	return response, nil
}
