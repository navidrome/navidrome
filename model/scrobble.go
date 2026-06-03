package model

import "time"

type Scrobble struct {
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
	Client         string
	Source         string
	Origin         string
	PlaybackMode   string
}

type ScrobbleRepository interface {
	RecordScrobble(mediaFileID string, submissionTime time.Time, client, source, origin, playbackMode string) error
}
