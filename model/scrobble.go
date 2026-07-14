package model

import "time"

type Scrobble struct {
	ID             int64  `structs:"id" json:"id"`
	MediaFileID    string `structs:"media_file_id" json:"mediaFileId"`
	UserID         string `json:"-"`
	SubmissionTime int64  `structs:"submission_time" json:"submissionTime"`
}

type ScrobbleRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Get(id string) (*Scrobble, error)
	GetAll(options ...QueryOptions) (Scrobbles, error)
	RecordScrobble(mediaFileID string, submissionTime time.Time) error
}

type Scrobbles []Scrobble
