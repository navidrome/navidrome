package dto

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

func TicksFromSeconds(sec float32) int64 { return int64(float64(sec) * 1e7) }

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
