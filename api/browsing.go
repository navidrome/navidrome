package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/utils"
)

type BrowsingController struct {
	browser engine.Browser
}

func NewBrowsingController(browser engine.Browser) *BrowsingController {
	return &BrowsingController{browser: browser}
}

func (c *BrowsingController) GetMusicFolders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mediaFolderList, _ := c.browser.MediaFolders()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.ID
		folders[i].Name = f.Name
	}
	response := NewResponse()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	return response, nil
}

func (c *BrowsingController) getArtistIndex(r *http.Request, ifModifiedSince time.Time) (*responses.Indexes, error) {
	indexes, lastModified, err := c.browser.Indexes(ifModifiedSince)
	if err != nil {
		log.Error(r, "Error retrieving Indexes", "error", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	res := &responses.Indexes{
		IgnoredArticles: conf.Sonic.IgnoredArticles,
		LastModified:    fmt.Sprint(utils.ToMillis(lastModified)),
	}

	res.Index = make([]responses.Index, len(indexes))
	for i, idx := range indexes {
		res.Index[i].Name = idx.ID
		res.Index[i].Artists = make([]responses.Artist, len(idx.Artists))
		for j, a := range idx.Artists {
			res.Index[i].Artists[j].Id = a.ArtistID
			res.Index[i].Artists[j].Name = a.Artist
			res.Index[i].Artists[j].AlbumCount = a.AlbumCount
		}
	}
	return res, nil
}

func (c *BrowsingController) GetIndexes(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ifModifiedSince := ParamTime(r, "ifModifiedSince", time.Time{})

	res, err := c.getArtistIndex(r, ifModifiedSince)
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Indexes = res
	return response, nil
}

func (c *BrowsingController) GetArtists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	res, err := c.getArtistIndex(r, time.Time{})
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Artist = res
	return response, nil
}

func (c *BrowsingController) GetMusicDirectory(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := ParamString(r, "id")
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
	id := ParamString(r, "id")
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
	id := ParamString(r, "id")
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
	id := ParamString(r, "id")
	song, err := c.browser.GetSong(id)
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
	genres, err := c.browser.GetGenres()
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Genres = ToGenres(genres)
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
