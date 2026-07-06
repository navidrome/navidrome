package dto

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

func TicksFromSeconds(sec float32) int64 { return int64(float64(sec) * 1e7) }

// MediaSourceFromMediaFile builds the MediaSourceInfo describing direct playback of mf's
// source file. Shared by SongToBaseItem and getPlaybackInfo so clients see the same Size and
// Bitrate whether they read it from a browse response or from /PlaybackInfo -- Finamp's
// download dialog reads MediaSources[0].Size from the former.
func MediaSourceFromMediaFile(mf model.MediaFile) MediaSourceInfo {
	return MediaSourceInfo{
		Id:                   mf.ID,
		Protocol:             "Http",
		Container:            mf.Suffix,
		Size:                 mf.Size,
		Name:                 mf.Title,
		Type:                 "Default",
		RunTimeTicks:         TicksFromSeconds(mf.Duration),
		Bitrate:              mf.BitRate * 1000, // Navidrome stores kbps; Jellyfin's Bitrate is bps.
		SupportsDirectPlay:   true,
		SupportsDirectStream: true,
		SupportsTranscoding:  true,
		IsRemote:             false,
		SupportsProbing:      true,
		MediaStreams:         []any{},
		MediaAttachments:     []any{},
		Formats:              []string{},
	}
}

func UserData(a model.Annotations, itemID string) *UserItemDataDto {
	d := &UserItemDataDto{
		PlayCount:  int(a.PlayCount),
		IsFavorite: a.Starred,
		Played:     a.PlayCount > 0,
		Key:        itemID,
		ItemId:     itemID,
	}
	if a.Rating > 0 {
		r := float64(a.Rating) * 2 // Navidrome 0-5 -> Jellyfin 0-10
		d.Rating = &r
	}
	if a.PlayDate != nil {
		s := a.PlayDate.UTC().Format(time.RFC3339)
		d.LastPlayedDate = &s
	}
	return d
}

func SongToBaseItem(mf model.MediaFile) BaseItemDto {
	item := BaseItemDto{
		Name:              mf.Title,
		Id:                mf.ID,
		Type:              "Audio",
		MediaType:         "Audio",
		IsFolder:          false,
		ParentId:          mf.AlbumID,
		Album:             mf.Album,
		AlbumId:           mf.AlbumID,
		AlbumArtist:       mf.AlbumArtist,
		Artists:           []string{mf.Artist},
		RunTimeTicks:      TicksFromSeconds(mf.Duration),
		Container:         mf.Suffix,
		CanDownload:       true,
		BackdropImageTags: []string{},
		UserData:          UserData(mf.Annotations, mf.ID),
		MediaSources:      []MediaSourceInfo{MediaSourceFromMediaFile(mf)},
	}
	if mf.Year > 0 {
		item.ProductionYear = new(mf.Year)
	}
	if mf.TrackNumber > 0 {
		item.IndexNumber = new(mf.TrackNumber)
	}
	if mf.DiscNumber > 0 {
		item.ParentIndexNumber = new(mf.DiscNumber)
	}
	if len(mf.Genres) > 0 {
		for _, g := range mf.Genres {
			item.Genres = append(item.Genres, g.Name)
		}
	} else if mf.Genre != "" {
		item.Genres = []string{mf.Genre}
	}
	// Finamp resolves song art via AlbumId + a non-empty AlbumPrimaryImageTag.
	if mf.AlbumID != "" {
		item.AlbumPrimaryImageTag = mf.AlbumID
	}
	return item
}

func AlbumToBaseItem(al model.Album) BaseItemDto {
	item := BaseItemDto{
		Name:              al.Name,
		Id:                al.ID,
		Type:              "MusicAlbum",
		IsFolder:          true,
		ParentId:          al.AlbumArtistID,
		AlbumArtist:       al.AlbumArtist,
		Album:             al.Name,
		ChildCount:        new(al.SongCount),
		SongCount:         new(al.SongCount),
		RunTimeTicks:      TicksFromSeconds(al.Duration),
		ImageTags:         map[string]string{"Primary": al.ID},
		BackdropImageTags: []string{},
		UserData:          UserData(al.Annotations, al.ID),
	}
	if al.AlbumArtistID != "" {
		item.AlbumArtists = []NameGuidPair{{Name: al.AlbumArtist, Id: al.AlbumArtistID}}
		item.ArtistItems = item.AlbumArtists
	}
	if al.MaxYear > 0 {
		item.ProductionYear = new(al.MaxYear)
	}
	if len(al.Genres) > 0 {
		for _, g := range al.Genres {
			item.Genres = append(item.Genres, g.Name)
		}
	}
	return item
}

func ArtistToBaseItem(ar model.Artist) BaseItemDto {
	return BaseItemDto{
		Name:              ar.Name,
		Id:                ar.ID,
		Type:              "MusicArtist",
		IsFolder:          true,
		AlbumCount:        new(ar.AlbumCount),
		SongCount:         new(ar.SongCount),
		ImageTags:         map[string]string{"Primary": ar.ID},
		BackdropImageTags: []string{},
		UserData:          UserData(ar.Annotations, ar.ID),
	}
}

func GenreToBaseItem(g model.Genre) BaseItemDto {
	return BaseItemDto{
		Name:              g.Name,
		Id:                g.ID,
		Type:              "MusicGenre",
		IsFolder:          true,
		BackdropImageTags: []string{},
	}
}

// PlaylistToBaseItem maps a playlist to a Playlist BaseItemDto. Unlike songs/albums/artists,
// model.Playlist has no embedded Annotations (no starred/rating/play-count), so UserData is
// left nil rather than synthesized.
func PlaylistToBaseItem(p model.Playlist) BaseItemDto {
	return BaseItemDto{
		Name:              p.Name,
		Id:                p.ID,
		Type:              "Playlist",
		IsFolder:          true,
		MediaType:         "Audio",
		ChildCount:        new(p.SongCount),
		RunTimeTicks:      TicksFromSeconds(p.Duration),
		BackdropImageTags: []string{},
	}
}
