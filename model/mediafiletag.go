package model

import "time"

type MediaFileTag struct {
	UserID      string    `structs:"user_id"       json:"userId"`
	MediaFileID string    `structs:"media_file_id" json:"mediaFileId"`
	TagName     string    `structs:"tag_name"      json:"tagName"`
	CreatedAt   time.Time `structs:"created_at"    json:"createdAt"`
}

type MediaFileTagRepository interface {
	TagSong(mediaFileID, tagName string) error
	UntagSong(mediaFileID, tagName string) error
	TagsForSong(mediaFileID string) ([]string, error)
	AllTagNames() ([]string, error)
	SongIDsForTag(tagName string) ([]string, error)
}
