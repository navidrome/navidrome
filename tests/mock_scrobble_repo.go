package tests

import (
	"context"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type MockScrobbleRepo struct {
	RecordedScrobbles []model.Scrobble
	ctx               context.Context
}

func (m *MockScrobbleRepo) Get(id string) (*model.Scrobble, error) {
	for idx := range m.RecordedScrobbles {
		if strconv.FormatInt(m.RecordedScrobbles[idx].ID, 10) == id {
			return &m.RecordedScrobbles[idx], nil
		}
	}

	return nil, model.ErrNotFound
}

func (m *MockScrobbleRepo) GetAll(options ...model.QueryOptions) (model.Scrobbles, error) {
	return m.RecordedScrobbles, nil
}

func (m *MockScrobbleRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	return int64(len(m.RecordedScrobbles)), nil
}

func (m *MockScrobbleRepo) RecordScrobble(fileID string, submissionTime time.Time) error {
	user, _ := request.UserFrom(m.ctx)
	m.RecordedScrobbles = append(m.RecordedScrobbles, model.Scrobble{
		MediaFileID:    fileID,
		UserID:         user.ID,
		SubmissionTime: submissionTime.Unix(),
	})
	return nil
}

var _ model.ScrobbleRepository = (*MockScrobbleRepo)(nil)
