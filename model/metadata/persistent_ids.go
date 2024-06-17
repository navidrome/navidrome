package metadata

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/str"
)

// FIXME Must be configurable
func (md Metadata) trackPID(mf model.MediaFile) string {
	value := cmp.Or(
		mf.MbzReleaseTrackID,
		fmt.Sprintf("%s\\%02d\\%02d", md.albumID(), mf.DiscNumber, mf.TrackNumber),
		fmt.Sprintf("%s\\%s", md.albumID(), md.mapTrackTitle()),
	)

	return fmt.Sprintf("%x", md5.Sum([]byte(str.Clear(strings.ToLower(value)))))
}

// FIXME Must be configurable
func (md Metadata) albumID() string {
	parts := []string{
		strings.ToLower(md.mapAlbumName()),
		strings.ToLower(md.String(AlbumVersion)),
		md.String(ReleaseDate),
	}
	albumPath := strings.Join(parts, "\\")
	return fmt.Sprintf("%x", md5.Sum([]byte(str.Clear(albumPath))))
}

// FIXME Must be configurable
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
