package metadata

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

// These legacy ID functions hash the same inputs as the original Navidrome ID generation,
// now emitted in the canonical base62 encoding (matching what the uniform-ids migration stores).

func legacyTrackID(mf model.MediaFile, prependLibId bool) string {
	key := mf.Path
	if prependLibId && mf.LibraryID != model.DefaultLibraryID {
		key = fmt.Sprintf("%d\\%s", mf.LibraryID, key)
	}
	sum := md5.Sum([]byte(key))
	return id.Encode128(sum)
}

func legacyAlbumID(mf model.MediaFile, md Metadata, prependLibId bool) string {
	_, _, releaseDate := md.mapDates()
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", legacyMapAlbumArtistName(md), legacyMapAlbumName(md)))
	if !conf.Server.Scanner.GroupAlbumReleases {
		if len(releaseDate) != 0 {
			albumPath = fmt.Sprintf("%s\\%s", albumPath, releaseDate)
		}
	}
	if prependLibId && mf.LibraryID != model.DefaultLibraryID {
		albumPath = fmt.Sprintf("%d\\%s", mf.LibraryID, albumPath)
	}
	sum := md5.Sum([]byte(albumPath))
	return id.Encode128(sum)
}

func legacyMapAlbumArtistName(md Metadata) string {
	values := []string{
		md.String(model.TagAlbumArtist),
		"",
		md.String(model.TagTrackArtist),
		consts.UnknownArtist,
	}
	if md.Bool(model.TagCompilation) {
		values[1] = consts.VariousArtists
	}
	return cmp.Or(values...)
}

func legacyMapAlbumName(md Metadata) string {
	return cmp.Or(
		md.String(model.TagAlbum),
		consts.UnknownAlbum,
	)
}
