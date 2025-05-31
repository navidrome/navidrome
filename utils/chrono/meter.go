package chrono

import (
	"time"

	. "github.com/navidrome/navidrome/utils/gg"
)

// Meter is a simple stopwatch
type Meter struct {
	elapsed time.Duration
	mark    *time.Time
}

func (m *Meter) Start() {
	m.mark = P(time.Now())
}

func (m *Meter) Stop() time.Duration {
	if m.mark == nil {
		return m.elapsed
	}
	m.elapsed += time.Since(*m.mark)
	m.mark = nil
	return m.elapsed
}

func (m *Meter) Elapsed() time.Duration {
	elapsed := m.elapsed
	if m.mark != nil {
		elapsed += time.Since(*m.mark)
	}
	return elapsed
}
