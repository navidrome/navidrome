package model

import "time"

type Scrobble struct {
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
}

type HistoryEntry struct {
	MediaFile
	PlayedAt time.Time `json:"playedAt"`
}

type ScrobbleRepository interface {
	RecordScrobble(mediaFileID string, submissionTime time.Time) error
	GetHistory(offset, count int) ([]HistoryEntry, error)
}
