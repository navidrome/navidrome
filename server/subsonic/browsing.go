package subsonic

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/engine"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type BrowsingController struct {
	browser engine.Browser
	ds      model.DataStore
}

func NewBrowsingController(browser engine.Browser, ds model.DataStore) *BrowsingController {
	return &BrowsingController{browser: browser, ds: ds}
}

func (c *BrowsingController) GetMusicFolders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mediaFolderList, _ := c.ds.MediaFolder(r.Context()).GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.ID
		folders[i].Name = f.Name
	}
	response := NewResponse()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	return response, nil
}

func (c *BrowsingController) getArtistIndex(ctx context.Context, mediaFolderId string, ifModifiedSince time.Time) (*responses.Indexes, error) {
	folder, err := c.ds.MediaFolder(ctx).Get(mediaFolderId)
	if err != nil {
		log.Error(ctx, "Error retrieving MediaFolder", "id", mediaFolderId, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	l, err := c.ds.Property(ctx).DefaultGet(model.PropLastScan+"-"+folder.Path, "-1")
	if err != nil {
		log.Error(ctx, "Error retrieving LastScan property", err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	var indexes model.ArtistIndexes
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)
	if lastModified.After(ifModifiedSince) {
		indexes, err = c.ds.Artist(ctx).GetIndex()
		if err != nil {
			log.Error(ctx, "Error retrieving Indexes", err)
			return nil, NewError(responses.ErrorGeneric, "Internal Error")
		}
	}

	res := &responses.Indexes{
		IgnoredArticles: conf.Server.IgnoredArticles,
		LastModified:    utils.ToMillis(lastModified),
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

	res, err := c.getArtistIndex(r.Context(), musicFolderId, ifModifiedSince)
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Indexes = res
	return response, nil
}

func (c *BrowsingController) GetArtists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamString(r, "musicFolderId")
	res, err := c.getArtistIndex(r.Context(), musicFolderId, time.Time{})
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
	response.Directory = c.buildDirectory(r.Context(), dir)
	return response, nil
}

func (c *BrowsingController) GetArtist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	artist, err := c.ds.Artist(ctx).Get(id)
	switch {
	case err == model.ErrNotFound:
		log.Error(ctx, "Requested ArtistID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Artist not found")
	case err != nil:
		log.Error(ctx, "Error retrieving artist", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	albums, err := c.ds.Album(ctx).FindByArtist(id)
	if err != nil {
		log.Error(ctx, "Error retrieving albums by artist", "id", id, "name", artist.Name, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.ArtistWithAlbumsID3 = c.buildArtist(ctx, artist, albums)
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
	response.AlbumWithSongsID3 = c.buildAlbum(r.Context(), dir)
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
	child := ToChild(r.Context(), *song)
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

const placeholderArtistImageSmallUrl = "https://lastfm.freetls.fastly.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageMediumUrl = "https://lastfm.freetls.fastly.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png"
const placeholderArtistImageLargeUrl = "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png"

// TODO Integrate with Last.FM
func (c *BrowsingController) GetArtistInfo(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.ArtistInfo = &responses.ArtistInfo{}
	response.ArtistInfo.Biography = "Biography not available"
	response.ArtistInfo.SmallImageUrl = placeholderArtistImageSmallUrl
	response.ArtistInfo.MediumImageUrl = placeholderArtistImageMediumUrl
	response.ArtistInfo.LargeImageUrl = placeholderArtistImageLargeUrl
	return response, nil
}

// TODO Integrate with Last.FM
func (c *BrowsingController) GetArtistInfo2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.ArtistInfo2 = &responses.ArtistInfo2{}
	response.ArtistInfo2.Biography = "Biography not available"
	response.ArtistInfo2.SmallImageUrl = placeholderArtistImageSmallUrl
	response.ArtistInfo2.MediumImageUrl = placeholderArtistImageSmallUrl
	response.ArtistInfo2.LargeImageUrl = placeholderArtistImageSmallUrl
	return response, nil
}

// TODO Integrate with Last.FM
func (c *BrowsingController) GetTopSongs(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.TopSongs = &responses.TopSongs{}
	return response, nil
}

func (c *BrowsingController) buildDirectory(ctx context.Context, d *engine.DirectoryInfo) *responses.Directory {
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

	dir.Child = ToChildren(ctx, d.Entries)
	return dir
}

func (c *BrowsingController) buildArtist(ctx context.Context, artist *model.Artist, albums model.Albums) *responses.ArtistWithAlbumsID3 {
	dir := &responses.ArtistWithAlbumsID3{}
	dir.Id = artist.ID
	dir.Name = artist.Name
	dir.AlbumCount = artist.AlbumCount
	if artist.Starred {
		dir.Starred = &artist.StarredAt
	}

	dir.Album = ChildrenFromAlbums(ctx, albums)
	return dir
}

func (c *BrowsingController) buildAlbum(ctx context.Context, d *engine.DirectoryInfo) *responses.AlbumWithSongsID3 {
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

	dir.Song = ToChildren(ctx, d.Entries)
	return dir
}
