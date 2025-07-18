package metadata

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

type hashFunc = func(...string) string

// createGetPID returns a function that calculates the persistent ID for a given spec, getting the referenced values from the metadata
// The spec is a pipe-separated list of fields, where each field is a comma-separated list of attributes
// Attributes can be either tags or some processed values like folder, albumid, albumartistid, etc.
// For each field, it gets all its attributes values and concatenates them, then hashes the result.
// If a field is empty, it is skipped and the function looks for the next field.
type getPIDFunc = func(mf model.MediaFile, md Metadata, spec string, prependLibId bool) string

func createGetPID(hash hashFunc) getPIDFunc {
	var getPID getPIDFunc
	getAttr := func(mf model.MediaFile, md Metadata, attr string, prependLibId bool) string {
		attr = strings.TrimSpace(strings.ToLower(attr))
		switch attr {
		case "albumid":
			return getPID(mf, md, conf.Server.PID.Album, prependLibId)
		case "folder":
			return filepath.Dir(mf.Path)
		case "albumartistid":
			return hash(str.Clear(strings.ToLower(mf.AlbumArtist)))
		case "title":
			return mf.Title
		case "album":
			return str.Clear(strings.ToLower(md.String(model.TagAlbum)))
		}
		return md.String(model.TagName(attr))
	}
	getPID = func(mf model.MediaFile, md Metadata, spec string, prependLibId bool) string {
		pid := ""
		fields := strings.Split(spec, "|")
		for _, field := range fields {
			attributes := strings.Split(field, ",")
			hasValue := false
			values := slice.Map(attributes, func(attr string) string {
				v := getAttr(mf, md, attr, prependLibId)
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
		if prependLibId {
			pid = fmt.Sprintf("%d\\%s", mf.LibraryID, pid)
		}
		return hash(pid)
	}

	return func(mf model.MediaFile, md Metadata, spec string, prependLibId bool) string {
		switch spec {
		case "track_legacy":
			return legacyTrackID(mf, prependLibId)
		case "album_legacy":
			return legacyAlbumID(mf, md, prependLibId)
		}
		return getPID(mf, md, spec, prependLibId)
	}
}

func (md Metadata) trackPID(mf model.MediaFile) string {
	return createGetPID(id.NewHash)(mf, md, conf.Server.PID.Track, true)
}

func (md Metadata) albumID(mf model.MediaFile, pidConf string) string {
	return createGetPID(id.NewHash)(mf, md, pidConf, true)
}

// BFR Must be configurable?
func (md Metadata) artistID(name string) string {
	mf := model.MediaFile{AlbumArtist: name}
	return createGetPID(id.NewHash)(mf, md, "albumartistid", false)
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
