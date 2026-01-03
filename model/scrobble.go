package model

import "time"

type Scrobble struct {
	ID             string
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
	Duration       *int // Duration in seconds the user actually listened. Nil if unknown.
}

type ScrobbleRepository interface {
	// RecordScrobble creates a new scrobble record and returns its ID
	RecordScrobble(mediaFileID string, submissionTime time.Time, duration *int) (string, error)
	// UpdateDuration updates the duration of an existing scrobble
	UpdateDuration(id string, duration int) error
}
