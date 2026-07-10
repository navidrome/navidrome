package model

import "time"

type Scrobble struct {
	MediaFileID string `json:"-"`
	UserID      string `json:"-"`

	SubmissionTime time.Time `structs:"submission_time" json:"submissionTime"`
	RowId          int64     `structs:"row_id" json:"rowId"`
	MediaFile
}

type ScrobbleRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Get(id string) (*Scrobble, error)
	GetAll(options ...QueryOptions) (Scrobbles, error)
	RecordScrobble(mediaFileID string, submissionTime time.Time) error
}

type Scrobbles []Scrobble
