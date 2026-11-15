package apiauth

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

func (s *Service) SetClock(now func() time.Time) { s.now = now }

func (s *Service) SetCheckers(f func(model.DataStore) []CredentialChecker) { s.checkers = f }
