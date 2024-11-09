package metadata

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

const (
	albumPIDSpec = "musicbrainz_albumid|albumartistid,album,version,releasedate"
)

type hashFunc = func(string) string

// getPID returns the persistent ID for a given spec, getting the referenced values from the metadata
// The spec is a pipe-separated list of fields, where each field is a comma-separated list of attributes
// For each field, it gets all its attributes values and concatenates them, then hashes the result.
// If a field is empty, it is skipped and the function looks for the next field.
func createGetPID(hash hashFunc) func(md Metadata, spec string) string {
	var getPID func(md Metadata, spec string, hash hashFunc) string
	getAttr := func(md Metadata, attr string) string {
		switch attr {
		case "albumid":
			return getPID(md, albumPIDSpec, hash)
		case "folder":
			return filepath.Dir(md.FilePath())
		}
		return md.String(model.TagName(attr))
	}
	getPID = func(md Metadata, spec string, hash hashFunc) string {
		pid := ""
		fields := strings.Split(spec, "|")
		for _, field := range fields {
			attributes := strings.Split(field, ",")
			hasValue := false
			values := slice.Map(attributes, func(attr string) string {
				v := getAttr(md, attr)
				if v != "" {
					hasValue = true
				}
				return v
			})
			if hasValue {
				pid += strings.Join(values, "\\")
				break
			}
		}
		return hash(pid)
	}

	return func(md Metadata, spec string) string {
		return getPID(md, spec, hash)
	}
}

// BFR Must be configurable
func (md Metadata) trackPID(mf model.MediaFile) string {
	value := cmp.Or(
		mf.MbzReleaseTrackID,
		fmt.Sprintf("%s\\%02d\\%02d%s",
			md.albumID(mf),
			mf.DiscNumber,
			mf.TrackNumber,
			md.mapTrackTitle(),
		),
	)

	return id.NewHash(str.Clear(strings.ToLower(value)))
}

// BFR Must be configurable
// BFR Tag mapper for PIDs need to use functions, to be able to extract attributes or processed values
func (md Metadata) albumID(mf model.MediaFile) string {
	value := cmp.Or(
		mf.MbzAlbumID,
		fmt.Sprintf("%s\\%s\\%s\\%s",
			md.artistID(mf.AlbumArtist),
			strings.ToLower(md.mapAlbumName()),
			strings.ToLower(md.String(model.TagAlbumVersion)),
			md.String(model.TagReleaseDate),
		),
	)
	return id.NewHash(str.Clear(strings.ToLower(value)))
}

// BFR Must be configurable?
func (md Metadata) artistID(name string) string {
	return id.NewHash(str.Clear(strings.ToLower(name)))
}

func (md Metadata) mapTrackTitle() string {
	if title := md.String(model.TagTitle); title != "" {
		return title
	}
	return utils.BaseName(md.FilePath())
}

func (md Metadata) mapAlbumName() string {
	return cmp.Or(
		md.String(model.TagAlbum),
		consts.UnknownAlbum,
	)
}
