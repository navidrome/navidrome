package metadata

import (
	"crypto/md5"
	"fmt"
	"path"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
)

func (md Metadata) trackPID(mf model.MediaFile) string {
	value := ""
	if mf.MbzReleaseTrackID != "" {
		value = mf.MbzReleaseTrackID
	} else if value == "" && mf.TrackNumber > 0 {
		value = fmt.Sprintf("%s\\%02d\\%02d", md.albumPID(), mf.DiscNumber, mf.TrackNumber)
	} else {
		value = fmt.Sprintf("%s\\%s", md.albumPID(), md.mapTrackTitle())
	}

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(value))))
}

func (md Metadata) albumPID() string {
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", md.mapAlbumArtistName(), md.mapAlbumName()))
	if !conf.Server.Scanner.GroupAlbumReleases {
		releaseDate := md.String(ReleaseDate)
		if releaseDate != "" {
			albumPath = fmt.Sprintf("%s\\%s", albumPath, releaseDate)
		}
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath)))
}

//nolint:unused
func (md Metadata) artistPID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(md.mapTrackArtistName()))))
}

//nolint:unused
func (md Metadata) albumArtistPID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(md.mapAlbumArtistName()))))
}

func (md Metadata) mapTrackTitle() string {
	if title := md.String(Title); title != "" {
		return title
	}
	s := md.FilePath()
	s = path.Base(s)
	e := path.Ext(s)
	return strings.TrimSuffix(s, e)
}

func (md Metadata) mapAlbumArtistName() string {
	if n := md.String(AlbumArtist); n != "" {
		return n
	}
	if md.Bool(Compilation) {
		return consts.VariousArtists
	}
	if n := md.String(TrackArtist); n != "" {
		return n
	}
	return consts.UnknownArtist
}

func (md Metadata) mapTrackArtistName() string {
	if n := md.String(TrackArtist); n != "" {
		return n
	}
	return consts.UnknownArtist
}

func (md Metadata) mapAlbumName() string {
	if n := md.String(Album); n != "" {
		return n
	}
	return consts.UnknownAlbum
}
