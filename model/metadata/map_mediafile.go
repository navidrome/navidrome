package metadata

import (
	"encoding/json"
	"maps"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (md Metadata) ToMediaFile(libID int, folderID string) model.MediaFile {
	if md.mediaFileJSON == "" {
		log.Warn("Missing media_file_json from Rust worker", "file", md.filePath)
		return md.shellMediaFile(libID, folderID)
	}
	mf, ok := md.mediaFileFromRust(libID, folderID)
	if !ok {
		return md.shellMediaFile(libID, folderID)
	}
	return mf
}

func (md Metadata) shellMediaFile(libID int, folderID string) model.MediaFile {
	mf := model.MediaFile{
		LibraryID: libID,
		FolderID:  folderID,
		Path:      md.FilePath(),
		Suffix:    md.Suffix(),
		Size:      md.Size(),
		BirthTime: md.BirthTime(),
		UpdatedAt: md.ModTime(),
		Tags:      maps.Clone(md.tags),
		Lyrics:    md.lyricsJSONOrEmpty(),
	}
	mf.HasCoverArt = md.HasPicture()
	mf.Duration = md.Length()
	mf.BitRate = md.AudioProperties().BitRate
	mf.SampleRate = md.AudioProperties().SampleRate
	if bd := md.AudioProperties().BitDepth; bd > 0 {
		mf.BitDepth = new(bd)
	}
	mf.Channels = md.AudioProperties().Channels
	mf.Codec = md.AudioProperties().Codec
	return mf
}

func (md Metadata) mediaFileFromRust(libID int, folderID string) (model.MediaFile, bool) {
	var payload struct {
		model.MediaFile
		Participants     map[string][]model.Participant `json:"participants"`
		Tags             map[string][]string            `json:"tags"`
		PID              string                         `json:"pid"`
		AlbumID          string                         `json:"albumId"`
		SearchNormalized string                         `json:"searchNormalized"`
	}
	if err := json.Unmarshal([]byte(md.mediaFileJSON), &payload); err != nil {
		log.Warn("Rust media_file_json decode failed", "file", md.filePath, err)
		return model.MediaFile{}, false
	}
	mf := payload.MediaFile
	// MediaFile.SearchNormalized is json:"-" for API payloads; accept the Rust scan field explicitly.
	if payload.SearchNormalized != "" {
		mf.SearchNormalized = payload.SearchNormalized
	}
	if len(payload.Participants) > 0 {
		mf.Participants = make(model.Participants, len(payload.Participants))
		for roleKey, list := range payload.Participants {
			role := model.RoleFromString(roleKey)
			if role == model.RoleInvalid {
				continue
			}
			for i := range list {
				if list[i].ID == "" {
					list[i].ID = md.artistID(list[i].Name)
				}
			}
			mf.Participants[role] = list
		}
	}
	mf.LibraryID = libID
	mf.FolderID = folderID
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.BirthTime = md.BirthTime()
	mf.UpdatedAt = md.ModTime()
	mf.HasCoverArt = md.HasPicture()
	mf.Duration = md.Length()
	mf.BitRate = md.AudioProperties().BitRate
	mf.SampleRate = md.AudioProperties().SampleRate
	if bd := md.AudioProperties().BitDepth; bd > 0 {
		mf.BitDepth = new(bd)
	}
	mf.Channels = md.AudioProperties().Channels
	mf.Codec = md.AudioProperties().Codec
	if mf.Lyrics == "" {
		mf.Lyrics = md.lyricsJSONOrEmpty()
	}
	if payload.PID != "" {
		mf.PID = payload.PID
	} else {
		mf.PID = md.trackPID(mf)
	}
	if payload.AlbumID != "" {
		mf.AlbumID = payload.AlbumID
	} else {
		mf.AlbumID = md.albumID(mf, conf.Server.PID.Album)
	}
	mf.ArtistID = mf.Participants.First(model.RoleArtist).ID
	mf.AlbumArtistID = mf.Participants.First(model.RoleAlbumArtist).ID
	mf.OrderArtistName = mf.Participants.First(model.RoleArtist).OrderArtistName
	mf.OrderAlbumArtistName = mf.Participants.First(model.RoleAlbumArtist).OrderArtistName
	mf.SortArtistName = mf.Participants.First(model.RoleArtist).SortArtistName
	mf.SortAlbumArtistName = mf.Participants.First(model.RoleAlbumArtist).SortArtistName
	mf.Tags = albumTagsFromRust(payload.Tags, md.tags)
	return mf, true
}

func albumTagsFromRust(rustTags map[string][]string, cleaned model.Tags) model.Tags {
	tags := mediaFileTagsFromCleaned(cleaned)
	if tags == nil {
		tags = model.Tags{}
	}
	if len(rustTags) == 0 {
		return tags
	}
	for key, values := range rustTags {
		tag := model.TagName(key)
		if len(values) == 0 {
			delete(tags, tag)
			continue
		}
		tags[tag] = values
	}
	return tags
}

// mediaFileTagsFromCleaned keeps additional tags and main tags marked album:true.
func mediaFileTagsFromCleaned(cleaned model.Tags) model.Tags {
	if len(cleaned) == 0 {
		return nil
	}
	tags := maps.Clone(cleaned)
	for tag, conf := range model.TagMainMappings() {
		if !conf.Album {
			delete(tags, tag)
		}
	}
	return tags
}

func (md Metadata) lyricsJSONOrEmpty() string {
	if md.lyricsJSON != "" {
		return md.lyricsJSON
	}
	return "[]"
}

func (md Metadata) AlbumID(mf model.MediaFile, pidConf string) string {
	return md.albumID(mf, pidConf)
}
