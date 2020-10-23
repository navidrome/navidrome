package subsonic

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type BrowsingController struct {
	ds model.DataStore
	ei core.ExternalInfo
}

func NewBrowsingController(ds model.DataStore, ei core.ExternalInfo) *BrowsingController {
	return &BrowsingController{ds: ds, ei: ei}
}

func (c *BrowsingController) GetMusicFolders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mediaFolderList, _ := c.ds.MediaFolder(r.Context()).GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.ID
		folders[i].Name = f.Name
	}
	response := newResponse()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	return response, nil
}

func (c *BrowsingController) getArtistIndex(ctx context.Context, mediaFolderId string, ifModifiedSince time.Time) (*responses.Indexes, error) {
	folder, err := c.ds.MediaFolder(ctx).Get(mediaFolderId)
	if err != nil {
		log.Error(ctx, "Error retrieving MediaFolder", "id", mediaFolderId, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	l, err := c.ds.Property(ctx).DefaultGet(model.PropLastScan+"-"+folder.Path, "-1")
	if err != nil {
		log.Error(ctx, "Error retrieving LastScan property", err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	var indexes model.ArtistIndexes
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)
	if lastModified.After(ifModifiedSince) {
		indexes, err = c.ds.Artist(ctx).GetIndex()
		if err != nil {
			log.Error(ctx, "Error retrieving Indexes", err)
			return nil, newError(responses.ErrorGeneric, "Internal Error")
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

	response := newResponse()
	response.Indexes = res
	return response, nil
}

func (c *BrowsingController) GetArtists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamString(r, "musicFolderId")
	res, err := c.getArtistIndex(r.Context(), musicFolderId, time.Time{})
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Artist = res
	return response, nil
}

func (c *BrowsingController) GetMusicDirectory(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	entity, err := core.GetEntityByID(ctx, c.ds, id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	var dir *responses.Directory

	switch v := entity.(type) {
	case *model.Artist:
		dir, err = c.buildArtistDirectory(ctx, v)
	case *model.Album:
		dir, err = c.buildAlbumDirectory(ctx, v)
	default:
		log.Error(r, "Requested ID of invalid type", "id", id, "entity", v)
		return nil, newError(responses.ErrorDataNotFound, "Directory not found")
	}

	if err != nil {
		log.Error(err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	response := newResponse()
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
		return nil, newError(responses.ErrorDataNotFound, "Artist not found")
	case err != nil:
		log.Error(ctx, "Error retrieving artist", "id", id, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	albums, err := c.ds.Album(ctx).FindByArtist(id)
	if err != nil {
		log.Error(ctx, "Error retrieving albums by artist", "id", id, "name", artist.Name, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	response := newResponse()
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
		return nil, newError(responses.ErrorDataNotFound, "Album not found")
	case err != nil:
		log.Error(ctx, "Error retrieving album", "id", id, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	mfs, err := c.ds.MediaFile(ctx).FindByAlbum(id)
	if err != nil {
		log.Error(ctx, "Error retrieving tracks from album", "id", id, "name", album.Name, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	response := newResponse()
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
		return nil, newError(responses.ErrorDataNotFound, "Song not found")
	case err != nil:
		log.Error(r, "Error retrieving MediaFile", "id", id, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}

	response := newResponse()
	child := childFromMediaFile(ctx, *mf)
	response.Song = &child
	return response, nil
}

func (c *BrowsingController) GetGenres(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	genres, err := c.ds.Genre(ctx).GetAll()
	if err != nil {
		log.Error(r, err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}
	for i, g := range genres {
		if strings.TrimSpace(g.Name) == "" {
			genres[i].Name = "<Empty>"
		}
	}
	sort.Slice(genres, func(i, j int) bool {
		return genres[i].Name < genres[j].Name
	})

	response := newResponse()
	response.Genres = toGenres(genres)
	return response, nil
}

func (c *BrowsingController) GetArtistInfo(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := requiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 20)
	includeNotPresent := utils.ParamBool(r, "includeNotPresent", false)

	info, err := c.ei.ArtistInfo(ctx, id, count, includeNotPresent)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.ArtistInfo = &responses.ArtistInfo{}
	response.ArtistInfo.Biography = info.Biography
	response.ArtistInfo.SmallImageUrl = info.SmallImageUrl
	response.ArtistInfo.MediumImageUrl = info.MediumImageUrl
	response.ArtistInfo.LargeImageUrl = info.LargeImageUrl
	response.ArtistInfo.LastFmUrl = info.LastFMUrl
	response.ArtistInfo.MusicBrainzID = info.MBID
	for _, s := range info.SimilarArtists {
		similar := responses.Artist{}
		similar.Id = s.ID
		similar.Name = s.Name
		similar.AlbumCount = s.AlbumCount
		if s.Starred {
			similar.Starred = &s.StarredAt
		}
		similar.UserRating = s.Rating
		response.ArtistInfo.SimilarArtist = append(response.ArtistInfo.SimilarArtist, similar)
	}
	return response, nil
}

func (c *BrowsingController) GetArtistInfo2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	info, err := c.GetArtistInfo(w, r)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.ArtistInfo2 = &responses.ArtistInfo2{}
	response.ArtistInfo2.ArtistInfoBase = info.ArtistInfo.ArtistInfoBase
	for _, s := range info.ArtistInfo.SimilarArtist {
		similar := responses.ArtistID3{}
		similar.Id = s.Id
		similar.Name = s.Name
		similar.AlbumCount = s.AlbumCount
		similar.Starred = s.Starred
		response.ArtistInfo2.SimilarArtist = append(response.ArtistInfo2.SimilarArtist, similar)
	}
	return response, nil
}

func (c *BrowsingController) GetSimilarSongs(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := requiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 50)

	songs, err := c.ei.SimilarSongs(ctx, id, count)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.SimilarSongs = &responses.SimilarSongs{
		Song: childrenFromMediaFiles(ctx, songs),
	}
	return response, nil
}

func (c *BrowsingController) GetSimilarSongs2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	res, err := c.GetSimilarSongs(w, r)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.SimilarSongs2 = &responses.SimilarSongs2{
		Song: res.SimilarSongs.Song,
	}
	return response, nil
}

func (c *BrowsingController) GetTopSongs(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	artist, err := requiredParamString(r, "artist", "artist parameter required")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 50)

	songs, err := c.ei.TopSongs(ctx, artist, count)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.TopSongs = &responses.TopSongs{
		Song: childrenFromMediaFiles(ctx, songs),
	}
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

	dir.Child = childrenFromAlbums(ctx, albums)
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

	dir.Album = childrenFromAlbums(ctx, albums)
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

	dir.Child = childrenFromMediaFiles(ctx, mfs)
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

	dir.Song = childrenFromMediaFiles(ctx, mfs)
	return dir
}
