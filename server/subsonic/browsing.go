package subsonic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type BrowsingController struct {
	browser engine.Browser
}

func NewBrowsingController(browser engine.Browser) *BrowsingController {
	return &BrowsingController{browser: browser}
}

func (c *BrowsingController) GetMusicFolders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mediaFolderList, _ := c.browser.MediaFolders(r.Context())
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.ID
		folders[i].Name = f.Name
	}
	response := NewResponse()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	return response, nil
}

func (c *BrowsingController) getArtistIndex(r *http.Request, musicFolderId string, ifModifiedSince time.Time) (*responses.Indexes, error) {
	indexes, lastModified, err := c.browser.Indexes(r.Context(), musicFolderId, ifModifiedSince)
	if err != nil {
		log.Error(r, "Error retrieving Indexes", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	res := &responses.Indexes{
		IgnoredArticles: conf.Server.IgnoredArticles,
		LastModified:    fmt.Sprint(utils.ToMillis(lastModified)),
	}

	res.Index = make([]responses.Index, len(indexes))
	for i, idx := range indexes {
		res.Index[i].Name = idx.ID
		res.Index[i].Artists = make([]responses.Artist, len(idx.Artists))
		for j, a := range idx.Artists {
			res.Index[i].Artists[j].Id = a.ID
			res.Index[i].Artists[j].Name = a.Name
			res.Index[i].Artists[j].AlbumCount = a.AlbumCount
		}
	}
	return res, nil
}

func (c *BrowsingController) GetIndexes(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamString(r, "musicFolderId")
	ifModifiedSince := utils.ParamTime(r, "ifModifiedSince", time.Time{})

	res, err := c.getArtistIndex(r, musicFolderId, ifModifiedSince)
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Indexes = res
	return response, nil
}

func (c *BrowsingController) GetArtists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamString(r, "musicFolderId")
	res, err := c.getArtistIndex(r, musicFolderId, time.Time{})
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Artist = res
	return response, nil
}

func (c *BrowsingController) GetMusicDirectory(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	dir, err := c.browser.Directory(r.Context(), id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Directory = c.buildDirectory(dir)
	return response, nil
}

func (c *BrowsingController) GetArtist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	dir, err := c.browser.Artist(r.Context(), id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ArtistID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Artist not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.ArtistWithAlbumsID3 = c.buildArtist(dir)
	return response, nil
}

func (c *BrowsingController) GetAlbum(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	dir, err := c.browser.Album(r.Context(), id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Album not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.AlbumWithSongsID3 = c.buildAlbum(dir)
	return response, nil
}

func (c *BrowsingController) GetSong(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	song, err := c.browser.GetSong(r.Context(), id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Song not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	child := ToChild(*song)
	response.Song = &child
	return response, nil
}

func (c *BrowsingController) GetGenres(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	genres, err := c.browser.GetGenres(r.Context())
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Genres = ToGenres(genres)
	return response, nil
}

const noImageAvailableUrl = "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ac/No_image_available.svg/1024px-No_image_available.svg.png"

// TODO Integrate with Last.FM
func (c *BrowsingController) GetArtistInfo(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.ArtistInfo = &responses.ArtistInfo{}
	response.ArtistInfo.Biography = "Biography not available"
	response.ArtistInfo.SmallImageUrl = noImageAvailableUrl
	response.ArtistInfo.MediumImageUrl = noImageAvailableUrl
	response.ArtistInfo.LargeImageUrl = noImageAvailableUrl
	return response, nil
}

// TODO Integrate with Last.FM
func (c *BrowsingController) GetArtistInfo2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.ArtistInfo2 = &responses.ArtistInfo2{}
	response.ArtistInfo2.Biography = "Biography not available"
	response.ArtistInfo2.SmallImageUrl = noImageAvailableUrl
	response.ArtistInfo2.MediumImageUrl = noImageAvailableUrl
	response.ArtistInfo2.LargeImageUrl = noImageAvailableUrl
	return response, nil
}

func (c *BrowsingController) buildDirectory(d *engine.DirectoryInfo) *responses.Directory {
	dir := &responses.Directory{
		Id:         d.Id,
		Name:       d.Name,
		Parent:     d.Parent,
		PlayCount:  d.PlayCount,
		AlbumCount: d.AlbumCount,
		UserRating: d.UserRating,
	}
	if !d.Starred.IsZero() {
		dir.Starred = &d.Starred
	}

	dir.Child = ToChildren(d.Entries)
	return dir
}

func (c *BrowsingController) buildArtist(d *engine.DirectoryInfo) *responses.ArtistWithAlbumsID3 {
	dir := &responses.ArtistWithAlbumsID3{}
	dir.Id = d.Id
	dir.Name = d.Name
	dir.AlbumCount = d.AlbumCount
	dir.CoverArt = d.CoverArt
	if !d.Starred.IsZero() {
		dir.Starred = &d.Starred
	}

	dir.Album = ToAlbums(d.Entries)
	return dir
}

func (c *BrowsingController) buildAlbum(d *engine.DirectoryInfo) *responses.AlbumWithSongsID3 {
	dir := &responses.AlbumWithSongsID3{}
	dir.Id = d.Id
	dir.Name = d.Name
	dir.Artist = d.Artist
	dir.ArtistId = d.ArtistId
	dir.CoverArt = d.CoverArt
	dir.SongCount = d.SongCount
	dir.Duration = d.Duration
	dir.PlayCount = d.PlayCount
	dir.Year = d.Year
	dir.Genre = d.Genre
	if !d.Created.IsZero() {
		dir.Created = &d.Created
	}
	if !d.Starred.IsZero() {
		dir.Starred = &d.Starred
	}

	dir.Song = ToChildren(d.Entries)
	return dir
}
