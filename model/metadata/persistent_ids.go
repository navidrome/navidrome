package metadata

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/str"
)

func (md Metadata) trackPID(mf model.MediaFile) string {
	value := cmp.Or(
		mf.MbzReleaseTrackID,
		fmt.Sprintf("%s\\%02d\\%02d", md.albumID(), mf.DiscNumber, mf.TrackNumber),
		fmt.Sprintf("%s\\%s", md.albumID(), md.mapTrackTitle()),
	)

	return fmt.Sprintf("%x", md5.Sum([]byte(str.Clear(strings.ToLower(value)))))
}

func (md Metadata) albumID() string {
	albumPath := strings.ToLower(md.mapAlbumName()) // FIXME
	if !conf.Server.Scanner.GroupAlbumReleases {
		releaseDate := md.String(ReleaseDate)
		if releaseDate != "" {
			albumPath = fmt.Sprintf("%s\\%s", albumPath, releaseDate)
		}
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(str.Clear(albumPath))))
}

func (md Metadata) artistID(name string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str.Clear(strings.ToLower(name)))))
}

func (md Metadata) mapTrackTitle() string {
	if title := md.String(Title); title != "" {
		return title
	}
	return utils.BaseName(md.FilePath())
}

func (md Metadata) mapAlbumName() string {
	return cmp.Or(
		md.String(Album),
		consts.UnknownAlbum,
	)
}
