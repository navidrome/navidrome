package model

import "time"

type Scrobble struct {
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
}

type ScrobbleRepository interface {
	RecordScrobble(mediaFileID, userID string, submissionTime time.Time) error
}
