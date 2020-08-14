package subsonic

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type BrowsingController struct {
	ds model.DataStore
}

func NewBrowsingController(ds model.DataStore) *BrowsingController {
	return &BrowsingController{ds: ds}
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
	ctx := r.Context()

	entity, err := getEntityByID(ctx, c.ds, id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	var dir *responses.Directory

	switch v := entity.(type) {
	case *model.Artist:
		dir, err = c.buildArtistDirectory(ctx, v)
	case *model.Album:
		dir, err = c.buildAlbumDirectory(ctx, v)
	default:
		log.Error(r, "Requested ID of invalid type", "id", id, "entity", v)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	}

	if err != nil {
		log.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Directory = dir
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
	ctx := r.Context()

	album, err := c.ds.Album(ctx).Get(id)
	switch {
	case err == model.ErrNotFound:
		log.Error(ctx, "Requested AlbumID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Album not found")
	case err != nil:
		log.Error(ctx, "Error retrieving album", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	mfs, err := c.ds.MediaFile(ctx).FindByAlbum(id)
	if err != nil {
		log.Error(ctx, "Error retrieving tracks from album", "id", id, "name", album.Name, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.AlbumWithSongsID3 = c.buildAlbum(ctx, album, mfs)
	return response, nil
}

func (c *BrowsingController) GetSong(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	mf, err := c.ds.MediaFile(ctx).Get(id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested MediaFileID not found ", "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Song not found")
	case err != nil:
		log.Error(r, "Error retrieving MediaFile", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	child := ChildFromMediaFile(ctx, *mf)
	response.Song = &child
	return response, nil
}

func (c *BrowsingController) GetGenres(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	genres, err := c.ds.Genre(ctx).GetAll()
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	for i, g := range genres {
		if strings.TrimSpace(g.Name) == "" {
			genres[i].Name = "<Empty>"
		}
	}
	sort.Slice(genres, func(i, j int) bool {
		return genres[i].Name < genres[j].Name
	})

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

func (c *BrowsingController) buildArtistDirectory(ctx context.Context, artist *model.Artist) (*responses.Directory, error) {
	dir := &responses.Directory{}
	dir.Id = artist.ID
	dir.Name = artist.Name
	dir.PlayCount = artist.PlayCount
	dir.AlbumCount = artist.AlbumCount
	dir.UserRating = artist.Rating
	if artist.Starred {
		dir.Starred = &artist.StarredAt
	}

	albums, err := c.ds.Album(ctx).FindByArtist(artist.ID)
	if err != nil {
		return nil, err
	}

	dir.Child = ChildrenFromAlbums(ctx, albums)
	return dir, nil
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

func (c *BrowsingController) buildAlbumDirectory(ctx context.Context, album *model.Album) (*responses.Directory, error) {
	dir := &responses.Directory{}
	dir.Id = album.ID
	dir.Name = album.Name
	dir.Parent = album.AlbumArtistID
	dir.PlayCount = album.PlayCount
	dir.UserRating = album.Rating
	dir.SongCount = album.SongCount
	dir.CoverArt = album.CoverArtId
	if album.Starred {
		dir.Starred = &album.StarredAt
	}

	mfs, err := c.ds.MediaFile(ctx).FindByAlbum(album.ID)
	if err != nil {
		return nil, err
	}

	dir.Child = ChildrenFromMediaFiles(ctx, mfs)
	return dir, nil
}

func (c *BrowsingController) buildAlbum(ctx context.Context, album *model.Album, mfs model.MediaFiles) *responses.AlbumWithSongsID3 {
	dir := &responses.AlbumWithSongsID3{}
	dir.Id = album.ID
	dir.Name = album.Name
	dir.Artist = album.AlbumArtist
	dir.ArtistId = album.AlbumArtistID
	dir.CoverArt = album.CoverArtId
	dir.SongCount = album.SongCount
	dir.Duration = int(album.Duration)
	dir.PlayCount = album.PlayCount
	dir.Year = album.MaxYear
	dir.Genre = album.Genre
	if !album.CreatedAt.IsZero() {
		dir.Created = &album.CreatedAt
	}
	if album.Starred {
		dir.Starred = &album.StarredAt
	}

	dir.Song = ChildrenFromMediaFiles(ctx, mfs)
	return dir
}
