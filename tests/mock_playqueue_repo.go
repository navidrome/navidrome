package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
)

type MockPlayQueueRepo struct {
	model.PlayQueueRepository
	Queue *model.PlayQueue
	Err   bool
}

func (m *MockPlayQueueRepo) Store(q *model.PlayQueue) error {
	if m.Err {
		return errors.New("error")
	}
	copyItems := make(model.MediaFiles, len(q.Items))
	copy(copyItems, q.Items)
	qCopy := *q
	qCopy.Items = copyItems
	m.Queue = &qCopy
	return nil
}

func (m *MockPlayQueueRepo) Retrieve(userId string) (*model.PlayQueue, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.Queue == nil || m.Queue.UserID != userId {
		return nil, model.ErrNotFound
	}
	copyItems := make(model.MediaFiles, len(m.Queue.Items))
	copy(copyItems, m.Queue.Items)
	qCopy := *m.Queue
	qCopy.Items = copyItems
	return &qCopy, nil
}
