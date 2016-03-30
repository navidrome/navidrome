package api

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/conf"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type BrowsingController struct {
	BaseAPIController
	browser engine.Browser
}

func (c *BrowsingController) Prepare() {
	utils.ResolveDependencies(&c.browser)
}

func (c *BrowsingController) GetMusicFolders() {
	mediaFolderList, _ := c.browser.MediaFolders()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	response := c.NewEmpty()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	c.SendResponse(response)
}

func (c *BrowsingController) getArtistIndex(ifModifiedSince time.Time) responses.Indexes {
	indexes, lastModified, err := c.browser.Indexes(ifModifiedSince)
	if err != nil {
		beego.Error("Error retrieving Indexes:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	res := responses.Indexes{
		IgnoredArticles: conf.GoSonic.IgnoredArticles,
		LastModified:    fmt.Sprint(utils.ToMillis(lastModified)),
	}

	res.Index = make([]responses.Index, len(indexes))
	for i, idx := range indexes {
		res.Index[i].Name = idx.Id
		res.Index[i].Artists = make([]responses.Artist, len(idx.Artists))
		for j, a := range idx.Artists {
			res.Index[i].Artists[j].Id = a.ArtistId
			res.Index[i].Artists[j].Name = a.Artist
			res.Index[i].Artists[j].AlbumCount = a.AlbumCount
		}
	}
	return res
}

func (c *BrowsingController) GetIndexes() {
	ifModifiedSince := c.ParamTime("ifModifiedSince", time.Time{})

	res := c.getArtistIndex(ifModifiedSince)

	response := c.NewEmpty()
	response.Indexes = &res
	c.SendResponse(response)
}

func (c *BrowsingController) GetArtists() {
	res := c.getArtistIndex(time.Time{})

	response := c.NewEmpty()
	response.Artist = &res
	c.SendResponse(response)
}

func (c *BrowsingController) GetMusicDirectory() {
	id := c.RequiredParamString("id", "id parameter required")

	dir, err := c.browser.Directory(id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("Requested Id", id, "not found:", err)
		c.SendError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.Directory = c.buildDirectory(dir)
	c.SendResponse(response)
}

func (c *BrowsingController) GetArtist() {
	id := c.RequiredParamString("id", "id parameter required")

	dir, err := c.browser.Artist(id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("Requested ArtistId", id, "not found:", err)
		c.SendError(responses.ErrorDataNotFound, "Artist not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.ArtistWithAlbumsID3 = c.buildArtist(dir)
	c.SendResponse(response)
}

func (c *BrowsingController) GetAlbum() {
	id := c.RequiredParamString("id", "id parameter required")

	dir, err := c.browser.Album(id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("Requested AlbumId", id, "not found:", err)
		c.SendError(responses.ErrorDataNotFound, "Album not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.AlbumWithSongsID3 = c.buildAlbum(dir)
	c.SendResponse(response)
}

func (c *BrowsingController) GetSong() {
	id := c.RequiredParamString("id", "id parameter required")

	song, err := c.browser.GetSong(id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("Requested Id", id, "not found:", err)
		c.SendError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	child := c.ToChild(*song)
	response.Song = &child
	c.SendResponse(response)
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

	dir.Child = c.ToChildren(d.Entries)
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

	dir.Album = c.ToAlbums(d.Entries)
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

	dir.Song = c.ToChildren(d.Entries)
	return dir
}
