package metadata

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"path"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
)

func (md Metadata) trackPID(mf model.MediaFile) string {
	value := cmp.Or(
		mf.MbzReleaseTrackID,
		fmt.Sprintf("%s\\%02d\\%02d", md.albumPID(), mf.DiscNumber, mf.TrackNumber),
		fmt.Sprintf("%s\\%s", md.albumPID(), md.mapTrackTitle()),
	)

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
	s := path.Base(md.FilePath())
	return strings.TrimSuffix(s, path.Ext(s))
}

func (md Metadata) mapAlbumArtistName() string {
	return cmp.Or(
		md.String(AlbumArtist),
		func() string {
			if md.Bool(Compilation) {
				return consts.VariousArtists
			}
			return ""
		}(),
		md.String(TrackArtist),
		consts.UnknownArtist,
	)
}

func (md Metadata) mapTrackArtistName() string {
	return cmp.Or(
		md.String(TrackArtist),
		consts.UnknownArtist,
	)
}

func (md Metadata) mapAlbumName() string {
	return cmp.Or(
		md.String(Album),
		consts.UnknownAlbum,
	)
}
