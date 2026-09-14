package model

import (
	"context"
	"time"

	"github.com/deluan/rest"
)

type Scrobble struct {
	ID             int64  `structs:"id" json:"id"`
	MediaFileID    string `structs:"media_file_id" json:"mediaFileId"`
	UserID         string `json:"-"`
	SubmissionTime int64  `structs:"submission_time" json:"submissionTime"`
}

type ScrobbleRepository interface {
	rest.Repository[Scrobble]
	CountAll(ctx context.Context, options ...QueryOptions) (int64, error)
	Get(ctx context.Context, id string) (*Scrobble, error)
	GetAll(ctx context.Context, options ...QueryOptions) (Scrobbles, error)
	RecordScrobble(ctx context.Context, mediaFileID string, submissionTime time.Time) error
}

type Scrobbles []Scrobble
