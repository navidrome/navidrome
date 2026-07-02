package model

import "time"

type Scrobble struct {
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
}

type MostPlayedEntry struct {
	MediaFile
	PlayCount int `json:"playCount"`
}

type ScrobbleRepository interface {
	RecordScrobble(mediaFileID string, submissionTime time.Time) error
	GetMostPlayed(offset, count int) ([]MostPlayedEntry, error)
}
