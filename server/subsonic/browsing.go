package subsonic

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic/filter"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

func (api *Router) GetMusicFolders(r *http.Request) (*responses.Subsonic, error) {
	mediaFolderList, _ := api.ds.MediaFolder(r.Context()).GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.ID
		folders[i].Name = f.Name
	}
	response := newResponse()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	return response, nil
}

func (api *Router) getArtistIndex(r *http.Request, mediaFolderId int, ifModifiedSince time.Time) (*responses.Indexes, error) {
	ctx := r.Context()
	folder, err := api.ds.MediaFolder(ctx).Get(int32(mediaFolderId))
	if err != nil {
		log.Error(ctx, "Error retrieving MediaFolder", "id", mediaFolderId, err)
		return nil, err
	}

	l, err := api.ds.Property(ctx).DefaultGet(model.PropLastScan+"-"+folder.Path, "-1")
	if err != nil {
		log.Error(ctx, "Error retrieving LastScan property", err)
		return nil, err
	}

	var indexes model.ArtistIndexes
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)
	if lastModified.After(ifModifiedSince) {
		indexes, err = api.ds.Artist(ctx).GetIndex()
		if err != nil {
			log.Error(ctx, "Error retrieving Indexes", err)
			return nil, err
		}
	}

	res := &responses.Indexes{
		IgnoredArticles: conf.Server.IgnoredArticles,
		LastModified:    utils.ToMillis(lastModified),
	}

	res.Index = make([]responses.Index, len(indexes))
	for i, idx := range indexes {
		res.Index[i].Name = idx.ID
		res.Index[i].Artists = toArtists(r, idx.Artists)
	}
	return res, nil
}

func (api *Router) GetIndexes(r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamInt(r, "musicFolderId", 0)
	ifModifiedSince := utils.ParamTime(r, "ifModifiedSince", time.Time{})

	res, err := api.getArtistIndex(r, musicFolderId, ifModifiedSince)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Indexes = res
	return response, nil
}

func (api *Router) GetArtists(r *http.Request) (*responses.Subsonic, error) {
	musicFolderId := utils.ParamInt(r, "musicFolderId", 0)
	res, err := api.getArtistIndex(r, musicFolderId, time.Time{})
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Artist = res
	return response, nil
}

func (api *Router) GetMusicDirectory(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if errors.Is(err, model.ErrNotFound) {
		log.Error(r, "Requested ID not found ", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Directory not found")
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var dir *responses.Directory

	switch v := entity.(type) {
	case *model.Artist:
		dir, err = api.buildArtistDirectory(ctx, v)
	case *model.Album:
		dir, err = api.buildAlbumDirectory(ctx, v)
	default:
		log.Error(r, "Requested ID of invalid type", "id", id, "entity", v)
		return nil, newError(responses.ErrorDataNotFound, "Directory not found")
	}

	if err != nil {
		log.Error(err)
		return nil, err
	}

	response := newResponse()
	response.Directory = dir
	return response, nil
}

func (api *Router) GetArtist(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	artist, err := api.ds.Artist(ctx).Get(id)
	if errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, "Requested ArtistID not found ", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Artist not found")
	}
	if err != nil {
		log.Error(ctx, "Error retrieving artist", "id", id, err)
		return nil, err
	}

	response := newResponse()
	response.ArtistWithAlbumsID3, err = api.buildArtist(r, artist)
	if err != nil {
		log.Error(ctx, "Error retrieving albums by artist", "id", artist.ID, "name", artist.Name, err)
	}
	return response, err
}

func (api *Router) GetAlbum(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")

	ctx := r.Context()

	album, err := api.ds.Album(ctx).Get(id)
	if errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, "Requested AlbumID not found ", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Album not found")
	}
	if err != nil {
		log.Error(ctx, "Error retrieving album", "id", id, err)
		return nil, err
	}

	mfs, err := api.ds.MediaFile(ctx).GetAll(filter.SongsByAlbum(id))
	if err != nil {
		log.Error(ctx, "Error retrieving tracks from album", "id", id, "name", album.Name, err)
		return nil, err
	}

	response := newResponse()
	response.AlbumWithSongsID3 = api.buildAlbum(ctx, album, mfs)
	return response, nil
}

func (api *Router) GetAlbumInfo(r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")
	ctx := r.Context()

	if err != nil {
		return nil, err
	}

	album, err := api.externalMetadata.UpdateAlbumInfo(ctx, id)

	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.AlbumInfo = &responses.AlbumInfo{}
	response.AlbumInfo.Notes = album.Description
	response.AlbumInfo.SmallImageUrl = public.ImageURL(r, album.CoverArtID(), 150)
	response.AlbumInfo.MediumImageUrl = public.ImageURL(r, album.CoverArtID(), 300)
	response.AlbumInfo.LargeImageUrl = public.ImageURL(r, album.CoverArtID(), 600)

	response.AlbumInfo.LastFmUrl = album.ExternalUrl
	response.AlbumInfo.MusicBrainzID = album.MbzAlbumID

	return response, nil
}

func (api *Router) GetSong(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	ctx := r.Context()

	mf, err := api.ds.MediaFile(ctx).Get(id)
	if errors.Is(err, model.ErrNotFound) {
		log.Error(r, "Requested MediaFileID not found ", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Song not found")
	}
	if err != nil {
		log.Error(r, "Error retrieving MediaFile", "id", id, err)
		return nil, err
	}

	response := newResponse()
	child := childFromMediaFile(ctx, *mf)
	response.Song = &child
	return response, nil
}

func (api *Router) GetGenres(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	genres, err := api.ds.Genre(ctx).GetAll(model.QueryOptions{Sort: "song_count, album_count, name desc", Order: "desc"})
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	for i, g := range genres {
		if g.Name == "" {
			genres[i].Name = "<Empty>"
		}
	}

	response := newResponse()
	response.Genres = toGenres(genres)
	return response, nil
}

func (api *Router) GetArtistInfo(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 20)
	includeNotPresent := utils.ParamBool(r, "includeNotPresent", false)

	artist, err := api.externalMetadata.UpdateArtistInfo(ctx, id, count, includeNotPresent)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.ArtistInfo = &responses.ArtistInfo{}
	response.ArtistInfo.Biography = artist.Biography
	response.ArtistInfo.SmallImageUrl = public.ImageURL(r, artist.CoverArtID(), 150)
	response.ArtistInfo.MediumImageUrl = public.ImageURL(r, artist.CoverArtID(), 300)
	response.ArtistInfo.LargeImageUrl = public.ImageURL(r, artist.CoverArtID(), 600)
	response.ArtistInfo.LastFmUrl = artist.ExternalUrl
	response.ArtistInfo.MusicBrainzID = artist.MbzArtistID
	for _, s := range artist.SimilarArtists {
		similar := toArtist(r, s)
		response.ArtistInfo.SimilarArtist = append(response.ArtistInfo.SimilarArtist, similar)
	}
	return response, nil
}

func (api *Router) GetArtistInfo2(r *http.Request) (*responses.Subsonic, error) {
	info, err := api.GetArtistInfo(r)
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
		similar.UserRating = s.UserRating
		similar.CoverArt = s.CoverArt
		similar.ArtistImageUrl = s.ArtistImageUrl
		response.ArtistInfo2.SimilarArtist = append(response.ArtistInfo2.SimilarArtist, similar)
	}
	return response, nil
}

func (api *Router) GetSimilarSongs(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 50)

	songs, err := api.externalMetadata.SimilarSongs(ctx, id, count)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.SimilarSongs = &responses.SimilarSongs{
		Song: childrenFromMediaFiles(ctx, songs),
	}
	return response, nil
}

func (api *Router) GetSimilarSongs2(r *http.Request) (*responses.Subsonic, error) {
	res, err := api.GetSimilarSongs(r)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.SimilarSongs2 = &responses.SimilarSongs2{
		Song: res.SimilarSongs.Song,
	}
	return response, nil
}

func (api *Router) GetTopSongs(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	artist, err := requiredParamString(r, "artist")
	if err != nil {
		return nil, err
	}
	count := utils.ParamInt(r, "count", 50)

	songs, err := api.externalMetadata.TopSongs(ctx, artist, count)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.TopSongs = &responses.TopSongs{
		Song: childrenFromMediaFiles(ctx, songs),
	}
	return response, nil
}

func (api *Router) buildArtistDirectory(ctx context.Context, artist *model.Artist) (*responses.Directory, error) {
	dir := &responses.Directory{}
	dir.Id = artist.ID
	dir.Name = artist.Name
	dir.PlayCount = artist.PlayCount
	if artist.PlayCount > 0 {
		dir.Played = &artist.PlayDate
	}
	dir.AlbumCount = int32(artist.AlbumCount)
	dir.UserRating = int32(artist.Rating)
	if artist.Starred {
		dir.Starred = &artist.StarredAt
	}

	albums, err := api.ds.Album(ctx).GetAllWithoutGenres(filter.AlbumsByArtistID(artist.ID))
	if err != nil {
		return nil, err
	}

	dir.Child = childrenFromAlbums(ctx, albums)
	return dir, nil
}

func (api *Router) buildArtist(r *http.Request, artist *model.Artist) (*responses.ArtistWithAlbumsID3, error) {
	ctx := r.Context()
	a := &responses.ArtistWithAlbumsID3{}
	a.ArtistID3 = toArtistID3(r, *artist)

	albums, err := api.ds.Album(ctx).GetAllWithoutGenres(filter.AlbumsByArtistID(artist.ID))
	if err != nil {
		return nil, err
	}

	a.Album = childrenFromAlbums(r.Context(), albums)
	return a, nil
}

func (api *Router) buildAlbumDirectory(ctx context.Context, album *model.Album) (*responses.Directory, error) {
	dir := &responses.Directory{}
	dir.Id = album.ID
	dir.Name = album.Name
	dir.Parent = album.AlbumArtistID
	dir.PlayCount = album.PlayCount
	if album.PlayCount > 0 {
		dir.Played = &album.PlayDate
	}
	dir.UserRating = int32(album.Rating)
	dir.SongCount = int32(album.SongCount)
	dir.CoverArt = album.CoverArtID().String()
	if album.Starred {
		dir.Starred = &album.StarredAt
	}

	mfs, err := api.ds.MediaFile(ctx).GetAll(filter.SongsByAlbum(album.ID))
	if err != nil {
		return nil, err
	}

	dir.Child = childrenFromMediaFiles(ctx, mfs)
	return dir, nil
}

func (api *Router) buildAlbum(ctx context.Context, album *model.Album, mfs model.MediaFiles) *responses.AlbumWithSongsID3 {
	dir := &responses.AlbumWithSongsID3{}
	dir.Id = album.ID
	dir.Name = album.Name
	dir.Artist = album.AlbumArtist
	dir.ArtistId = album.AlbumArtistID
	dir.CoverArt = album.CoverArtID().String()
	dir.SongCount = int32(album.SongCount)
	dir.Duration = int32(album.Duration)
	dir.PlayCount = album.PlayCount
	if album.PlayCount > 0 {
		dir.Played = &album.PlayDate
	}
	dir.Year = int32(album.MaxYear)
	dir.Genre = album.Genre
	dir.Genres = itemGenresFromGenres(album.Genres)
	dir.UserRating = int32(album.Rating)
	if !album.CreatedAt.IsZero() {
		dir.Created = &album.CreatedAt
	}
	if album.Starred {
		dir.Starred = &album.StarredAt
	}
	dir.MusicBrainzId = album.MbzAlbumID
	dir.Song = childrenFromMediaFiles(ctx, mfs)
	return dir
}
