package radio

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidSession = errors.New("invalid radio metadata session")
	ErrInvalidStation = errors.New("invalid radio metadata station")
)

type Station struct {
	ID        string
	StreamURL string
}

type TitleUpdate struct {
	RadioID   string    `json:"radioId"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type StreamReader func(ctx context.Context, streamURL string, handleTitle func(string)) error

type TitlePublisher func(ctx context.Context, update TitleUpdate)

type MetadataManager struct {
	mu sync.Mutex

	reader  StreamReader
	publish TitlePublisher
	now     func() time.Time
	backoff func(attempt int) time.Duration

	sessions map[string]activeSession
	readers  map[string]*activeReader
}

type ManagerOption func(*MetadataManager)

type activeSession struct {
	radioID   string
	streamURL string
}

type activeReader struct {
	streamURL string
	ctx       context.Context
	cancel    context.CancelFunc
	radioRefs map[string]int
	lastTitle string
}

func NewMetadataManager(reader StreamReader, publisher TitlePublisher, opts ...ManagerOption) *MetadataManager {
	m := &MetadataManager{
		reader:   reader,
		publish:  publisher,
		now:      time.Now,
		backoff:  defaultBackoff,
		sessions: map[string]activeSession{},
		readers:  map[string]*activeReader{},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func WithNow(now func() time.Time) ManagerOption {
	return func(m *MetadataManager) {
		if now != nil {
			m.now = now
		}
	}
}

func WithRetryBackoff(backoff func(attempt int) time.Duration) ManagerOption {
	return func(m *MetadataManager) {
		if backoff != nil {
			m.backoff = backoff
		}
	}
}

func (m *MetadataManager) Start(ctx context.Context, sessionID string, station Station) error {
	if sessionID == "" {
		return ErrInvalidSession
	}
	if station.ID == "" || station.StreamURL == "" {
		return ErrInvalidStation
	}
	if m.reader == nil {
		return errors.New("missing radio metadata stream reader")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if current, ok := m.sessions[sessionID]; ok {
		if current.radioID == station.ID && current.streamURL == station.StreamURL {
			return nil
		}
		m.removeSessionLocked(sessionID)
	}

	reader := m.readers[station.StreamURL]
	if reader == nil {
		readerCtx, cancel := context.WithCancel(ctx)
		reader = &activeReader{
			streamURL: station.StreamURL,
			ctx:       readerCtx,
			cancel:    cancel,
			radioRefs: map[string]int{},
		}
		m.readers[station.StreamURL] = reader
		go m.runReader(reader)
	}

	reader.radioRefs[station.ID]++
	m.sessions[sessionID] = activeSession{radioID: station.ID, streamURL: station.StreamURL}
	return nil
}

func (m *MetadataManager) Stop(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeSessionLocked(sessionID)
}

func (m *MetadataManager) removeSessionLocked(sessionID string) {
	session, ok := m.sessions[sessionID]
	if !ok {
		return
	}
	delete(m.sessions, sessionID)

	reader := m.readers[session.streamURL]
	if reader == nil {
		return
	}

	if count := reader.radioRefs[session.radioID]; count <= 1 {
		delete(reader.radioRefs, session.radioID)
	} else {
		reader.radioRefs[session.radioID] = count - 1
	}

	if len(reader.radioRefs) == 0 {
		delete(m.readers, session.streamURL)
		reader.cancel()
	}
}

func (m *MetadataManager) runReader(reader *activeReader) {
	attempt := 0
	for {
		err := m.reader(reader.ctx, reader.streamURL, func(title string) {
			m.publishTitle(reader, title)
		})
		if reader.ctx.Err() != nil {
			return
		}
		if err == nil {
			attempt = 0
		} else {
			attempt++
		}

		delay := m.backoff(attempt)
		if delay <= 0 {
			continue
		}

		timer := time.NewTimer(delay)
		select {
		case <-reader.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (m *MetadataManager) publishTitle(reader *activeReader, title string) {
	m.mu.Lock()
	if reader.ctx.Err() != nil || title == "" || title == reader.lastTitle {
		m.mu.Unlock()
		return
	}

	reader.lastTitle = title
	radioIDs := make([]string, 0, len(reader.radioRefs))
	for radioID := range reader.radioRefs {
		radioIDs = append(radioIDs, radioID)
	}
	updatedAt := m.now()
	publish := m.publish
	m.mu.Unlock()

	if publish == nil {
		return
	}
	for _, radioID := range radioIDs {
		publish(reader.ctx, TitleUpdate{
			RadioID:   radioID,
			Title:     title,
			UpdatedAt: updatedAt,
		})
	}
}

func defaultBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Duration(attempt) * time.Second
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}
