package tests

import (
	"context"
	"strconv"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type MockScrobbleRepo struct {
	RecordedScrobbles []model.Scrobble
}

func (m *MockScrobbleRepo) Get(_ context.Context, id string) (*model.Scrobble, error) {
	for idx := range m.RecordedScrobbles {
		if strconv.FormatInt(m.RecordedScrobbles[idx].ID, 10) == id {
			return &m.RecordedScrobbles[idx], nil
		}
	}

	return nil, model.ErrNotFound
}

func (m *MockScrobbleRepo) GetAll(_ context.Context, _ ...model.QueryOptions) (model.Scrobbles, error) {
	return m.RecordedScrobbles, nil
}

func (m *MockScrobbleRepo) CountAll(_ context.Context, _ ...model.QueryOptions) (int64, error) {
	return int64(len(m.RecordedScrobbles)), nil
}

func (m *MockScrobbleRepo) RecordScrobble(ctx context.Context, fileID string, submissionTime time.Time) error {
	user, _ := request.UserFrom(ctx)
	m.RecordedScrobbles = append(m.RecordedScrobbles, model.Scrobble{
		MediaFileID:    fileID,
		UserID:         user.ID,
		SubmissionTime: submissionTime.Unix(),
	})
	return nil
}

func (m *MockScrobbleRepo) Count(ctx context.Context, _ ...rest.QueryOptions) (int64, error) {
	return m.CountAll(ctx)
}

func (m *MockScrobbleRepo) Read(ctx context.Context, id string) (*model.Scrobble, error) {
	return m.Get(ctx, id)
}

func (m *MockScrobbleRepo) ReadAll(ctx context.Context, _ ...rest.QueryOptions) ([]model.Scrobble, error) {
	return m.GetAll(ctx)
}

var _ model.ScrobbleRepository = (*MockScrobbleRepo)(nil)
