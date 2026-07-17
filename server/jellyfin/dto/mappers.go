package dto

import (
	"cmp"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/model"
)

func TicksFromSeconds(sec float32) int64 { return int64(float64(sec) * 1e7) }

// premiereDate converts a possibly partial date tag ("2007", "2007-02") into the ISO 8601
// PremiereDate clients parse, falling back to year; nil when neither exists.
func premiereDate(date string, year int) *string {
	d := date
	switch len(d) {
	case 4:
		d += "-01-01"
	case 7:
		d += "-01"
	case 10: // already yyyy-mm-dd
	default:
		if year <= 0 {
			return nil
		}
		d = fmt.Sprintf("%04d-01-01", year)
	}
	s := d + "T00:00:00Z"
	return &s
}

// jellyfinDate formats t as the ISO 8601 string clients expect, or "" for the zero time so the
// field is omitted rather than sent as a meaningless epoch.
func jellyfinDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// channelLayout maps a channel count to the label Jellyfin clients expect on a MediaStream.
func channelLayout(n int) string {
	switch n {
	case 1:
		return "mono"
	case 2:
		return "stereo"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	default:
		return ""
	}
}

// MediaSourceFromMediaFile builds the MediaSourceInfo for direct playback of mf's source file.
// Shared by SongToBaseItem and getPlaybackInfo so Size/Bitrate match across browse and /PlaybackInfo
// responses (Finamp's download dialog reads MediaSources[0].Size from the browse response).
func MediaSourceFromMediaFile(mf model.MediaFile) MediaSourceInfo {
	return MediaSourceInfo{
		Id:                   EncodeID(mf.ID),
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
		MediaStreams: []MediaStream{{
			Type:          "Audio",
			Index:         0,
			Codec:         mf.Codec,
			BitRate:       mf.BitRate * 1000, // Navidrome stores kbps; Jellyfin's BitRate is bps.
			Channels:      mf.Channels,
			SampleRate:    mf.SampleRate,
			ChannelLayout: channelLayout(mf.Channels),
		}},
		MediaAttachments: []any{},
		Formats:          []string{},
	}
}

func UserData(a model.Annotations, itemID string) *UserItemDataDto {
	// Callers pass the raw model id; encode here so Key/ItemId match the encoded Id on the BaseItemDto.
	encodedID := EncodeID(itemID)
	d := &UserItemDataDto{
		PlayCount:  int(a.PlayCount),
		IsFavorite: a.Starred,
		Played:     a.PlayCount > 0,
		Key:        encodedID,
		ItemId:     encodedID,
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

// SongToBaseItem maps a media file to an Audio BaseItemDto. MediaSources and SortName are attached
// only when the request's Fields asks for them, mirroring real Jellyfin (which omits both from a
// plain list response); a nil fields set means neither.
func SongToBaseItem(mf model.MediaFile, fields Fields) BaseItemDto {
	item := BaseItemDto{
		Name:              mf.Title,
		Id:                EncodeID(mf.ID),
		Type:              "Audio",
		MediaType:         "Audio",
		IsFolder:          false,
		LocationType:      "FileSystem",
		HasLyrics:         mf.Lyrics != "",
		ParentId:          EncodeID(mf.AlbumID),
		Album:             mf.Album,
		AlbumId:           EncodeID(mf.AlbumID),
		AlbumArtist:       mf.AlbumArtist,
		Artists:           []string{mf.Artist},
		RunTimeTicks:      TicksFromSeconds(mf.Duration),
		DateCreated:       jellyfinDate(&mf.CreatedAt),
		Container:         mf.Suffix,
		CanDownload:       true,
		BackdropImageTags: []string{},
		UserData:          UserData(mf.Annotations, mf.ID),
	}
	if fields.Has("MediaSources") {
		item.MediaSources = []MediaSourceInfo{MediaSourceFromMediaFile(mf)}
	}
	if fields.Has("SortName") {
		item.SortName = cmp.Or(mf.SortTitle, mf.OrderTitle, mf.Title)
	}
	// Finamp's Now Playing screen reads ArtistItems for the displayed artist (falling back to "Unknown
	// Artist" if absent), even though Artists carries the same name. ArtistItems is the track artist;
	// AlbumArtists the album artist.
	if mf.ArtistID != "" {
		item.ArtistItems = []NameGuidPair{{Name: mf.Artist, Id: EncodeID(mf.ArtistID)}}
	}
	if mf.AlbumArtistID != "" {
		item.AlbumArtists = []NameGuidPair{{Name: mf.AlbumArtist, Id: EncodeID(mf.AlbumArtistID)}}
	}
	if mf.Year > 0 {
		item.ProductionYear = new(mf.Year)
	}
	item.PremiereDate = premiereDate(mf.Date, mf.Year)
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
		item.ImageBlurHashes = map[string]map[string]string{"Primary": {mf.AlbumID: blurHash(mf.AlbumID)}}
	}
	return item
}

func AlbumToBaseItem(al model.Album) BaseItemDto {
	item := BaseItemDto{
		Name:              al.Name,
		Id:                EncodeID(al.ID),
		Type:              "MusicAlbum",
		IsFolder:          true,
		ParentId:          EncodeID(al.AlbumArtistID),
		AlbumArtist:       al.AlbumArtist,
		Album:             al.Name,
		ChildCount:        new(al.SongCount),
		SongCount:         new(al.SongCount),
		RunTimeTicks:      TicksFromSeconds(al.Duration),
		DateCreated:       jellyfinDate(&al.CreatedAt),
		ImageTags:         map[string]string{"Primary": al.ID},
		ImageBlurHashes:   map[string]map[string]string{"Primary": {al.ID: blurHash(al.ID)}},
		BackdropImageTags: []string{},
		UserData:          UserData(al.Annotations, al.ID),
	}
	if al.AlbumArtistID != "" {
		item.AlbumArtists = []NameGuidPair{{Name: al.AlbumArtist, Id: EncodeID(al.AlbumArtistID)}}
		item.ArtistItems = item.AlbumArtists
	}
	if al.MaxYear > 0 {
		item.ProductionYear = new(al.MaxYear)
	}
	item.PremiereDate = premiereDate(al.Date, al.MaxYear)
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
		Id:                EncodeID(ar.ID),
		Type:              "MusicArtist",
		IsFolder:          true,
		AlbumCount:        new(ar.AlbumCount),
		SongCount:         new(ar.SongCount),
		DateCreated:       jellyfinDate(ar.CreatedAt),
		ImageTags:         map[string]string{"Primary": ar.ID},
		ImageBlurHashes:   map[string]map[string]string{"Primary": {ar.ID: blurHash(ar.ID)}},
		BackdropImageTags: []string{},
		UserData:          UserData(ar.Annotations, ar.ID),
	}
}

func GenreToBaseItem(g model.Genre) BaseItemDto {
	return BaseItemDto{
		Name:              g.Name,
		Id:                EncodeID(g.ID),
		Type:              "MusicGenre",
		IsFolder:          true,
		BackdropImageTags: []string{},
	}
}

// PlaylistToBaseItem maps a playlist to a Playlist BaseItemDto.
func PlaylistToBaseItem(p model.Playlist) BaseItemDto {
	// Finamp caches covers keyed by blurHash, so the tag (and blurhash) must change with the cover.
	// UpdatedAt versions it (Put bumps it on upload); over-invalidation only costs a refetch.
	tag := fmt.Sprintf("%s-%x", p.ID, p.UpdatedAt.UnixMilli())
	return BaseItemDto{
		Name: p.Name,
		Id:   EncodeID(p.ID),
		Type: "Playlist",
		// Synthetic path: Jellify only surfaces playlists whose Path contains "data" (real Jellyfin
		// stores them under its data folder), so without this its Playlists tab hides them all.
		Path:              "/data/playlists/" + p.ID,
		IsFolder:          true,
		MediaType:         "Audio",
		ChildCount:        new(p.SongCount),
		RunTimeTicks:      TicksFromSeconds(p.Duration),
		ImageTags:         map[string]string{"Primary": tag},
		ImageBlurHashes:   map[string]map[string]string{"Primary": {tag: blurHash(tag)}},
		BackdropImageTags: []string{},
		UserData:          UserData(p.Annotations, p.ID),
	}
}
