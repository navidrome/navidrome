package tests

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
)

// MockAppPasswordRepo is a minimal in-memory stand-in for
// model.AppPasswordRepository. By default GetActiveForUser returns an empty
// slice, so the Subsonic auth fallback short-circuits cleanly without
// panicking on nil-method invocations of an unmocked interface.
//
// Tests that need richer behaviour can populate Active or override fields
// directly.
type MockAppPasswordRepo struct {
	model.AppPasswordRepository
	Active             model.AppPasswords
	GetActiveErr       error
	UpdateLastUsedAtFn func(id string)
}

func CreateMockAppPasswordRepo() *MockAppPasswordRepo {
	return &MockAppPasswordRepo{}
}

func (m *MockAppPasswordRepo) GetActiveForUser(ctx context.Context, userID string) (model.AppPasswords, error) {
	if m.GetActiveErr != nil {
		return nil, m.GetActiveErr
	}
	out := make(model.AppPasswords, 0, len(m.Active))
	for _, ap := range m.Active {
		if ap.UserID == userID {
			out = append(out, ap)
		}
	}
	return out, nil
}

func (m *MockAppPasswordRepo) UpdateLastUsedAt(ctx context.Context, id string) error {
	if m.UpdateLastUsedAtFn != nil {
		m.UpdateLastUsedAtFn(id)
	}
	return nil
}

func (m *MockAppPasswordRepo) ListForUser(ctx context.Context, userID string) ([]model.AppPasswordPublic, error) {
	return nil, nil
}

func (m *MockAppPasswordRepo) Create(ctx context.Context, userID, name string, expiresAt *time.Time) (string, *model.AppPassword, error) {
	ap := &model.AppPassword{ID: "mock", UserID: userID, Name: name, CreatedAt: time.Now(), ExpiresAt: expiresAt}
	m.Active = append(m.Active, *ap)
	return "secret", ap, nil
}

func (m *MockAppPasswordRepo) Delete(ctx context.Context, id, ownerUserID string) error {
	return nil
}
