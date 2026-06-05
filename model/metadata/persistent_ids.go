package metadata

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/str"
)

type hashFunc = func(...string) string

// computePID calculates the persistent ID for a given spec. The spec is a
// pipe-separated list of fields, where each field is a comma-separated list of
// attributes. Attributes can be either tags or processed values like folder,
// albumid, albumartistid, etc. For each field, it gets all its attribute values
// and concatenates them, then hashes the result. If a field is empty, it is
// skipped and the function looks for the next field.
//
// Taking hash as a parameter (instead of closing over it in a factory) keeps
// mf on the stack: closing over mf would force the whole ~1KB MediaFile to the
// heap on every call.
func computePID(mf model.MediaFile, md Metadata, spec string, prependLibId bool, hash hashFunc) string {
	switch spec {
	case "track_legacy":
		return legacyTrackID(mf, prependLibId)
	case "album_legacy":
		return legacyAlbumID(mf, md, prependLibId)
	}
	pid := ""
	fields := strings.SplitSeq(spec, "|")
	for field := range fields {
		attributes := strings.Split(field, ",")
		values := make([]string, len(attributes))
		hasValue := false
		for i, attr := range attributes {
			v := getPIDAttr(mf, md, attr, prependLibId, spec, hash)
			if v != "" {
				hasValue = true
			}
			values[i] = v
		}
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

func getPIDAttr(mf model.MediaFile, md Metadata, attr string, prependLibId bool, spec string, hash hashFunc) string {
	attr = strings.TrimSpace(strings.ToLower(attr))
	switch attr {
	case "albumid":
		if spec == conf.Server.PID.Album {
			log.Error("Recursive PID definition detected, ignoring `albumid`", "spec", spec)
			return ""
		}
		return computePID(mf, md, conf.Server.PID.Album, prependLibId, hash)
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

func (md Metadata) trackPID(mf model.MediaFile) string {
	return computePID(mf, md, conf.Server.PID.Track, true, id.NewHash)
}

func (md Metadata) albumID(mf model.MediaFile, pidConf string) string {
	return computePID(mf, md, pidConf, true, id.NewHash)
}

// BFR Must be configurable?
func (md Metadata) artistID(name string) string {
	mf := model.MediaFile{AlbumArtist: name}
	return computePID(mf, md, "albumartistid", false, id.NewHash)
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
