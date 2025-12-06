package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockScrobbleRepo struct {
	RecordedScrobbles []model.Scrobble
}

func (m *MockScrobbleRepo) RecordScrobble(fileID, userID string, submissionTime time.Time) error {
	m.RecordedScrobbles = append(m.RecordedScrobbles, model.Scrobble{
		MediaFileID:    fileID,
		UserID:         userID,
		SubmissionTime: submissionTime,
	})
	return nil
}
