package cast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	libcast "github.com/vishen/go-chromecast/cast"
	pb "github.com/vishen/go-chromecast/cast/proto"
)

const (
	defaultSenderID   = "sender-0"
	defaultReceiverID = "receiver-0"

	namespaceConn  = "urn:x-cast:com.google.cast.tp.connection"
	namespaceRecv  = "urn:x-cast:com.google.cast.receiver"
	namespaceMedia = "urn:x-cast:com.google.cast.media"

	defaultMediaReceiverAppID = "CC1AD845"
	requestTimeout            = 5 * time.Second
)

var (
	errDisconnected      = errors.New("cast connection disconnected")
	errSessionClosed     = errors.New("cast session closed")
	errLoadFailed        = errors.New("cast load failed")
	errMissingTransport  = errors.New("cast receiver transport id unavailable")
	errMissingMedia      = errors.New("cast media session unavailable")
	errUnexpectedPlaying = errors.New("cast receiver remained playing after autoplay=false load")
)

type session interface {
	Snapshot() sessionSnapshot
	Events() <-chan sessionEvent
	Play(context.Context) error
	Pause(context.Context) error
	SeekTo(context.Context, float32, *string) error
	SetVolume(context.Context, float32) error
	Close() error
}

type connFactory func() libcast.Conn

type castSession struct {
	target Target
	conn   libcast.Conn

	cmdMu    sync.Mutex
	eventsMu sync.RWMutex
	mu       sync.RWMutex

	nextRequestID int
	waiters       map[int]chan protocolMessage
	snapshot      sessionSnapshot
	events        chan sessionEvent
	stateChanged  chan struct{}
	terminal      chan sessionEvent
	closeOnce     sync.Once
	done          chan struct{}
	connStarted   bool
	closing       bool
	eventsClosed  bool
}

type protocolMessage struct {
	Type           string
	RequestID      int
	ReceiverStatus *libcast.ReceiverStatusResponse
	MediaStatus    *libcast.MediaStatusResponse
	Err            error
}

type payloadHeader struct {
	Type      string `json:"type"`
	RequestID int    `json:"requestId,omitempty"`
}

func (p *payloadHeader) SetRequestId(id int) { p.RequestID = id }

type connectPayload struct{ payloadHeader }

type launchPayload struct {
	payloadHeader
	AppID string `json:"appId"`
}

type getStatusPayload struct{ payloadHeader }

type playPausePayload struct {
	payloadHeader
	MediaSessionID int `json:"mediaSessionId"`
}

type stopPayload struct {
	payloadHeader
	MediaSessionID int `json:"mediaSessionId"`
}

type seekPayload struct {
	payloadHeader
	MediaSessionID int     `json:"mediaSessionId"`
	CurrentTime    float32 `json:"currentTime"`
	ResumeState    *string `json:"resumeState,omitempty"`
}

type receiverVolumePayload struct {
	payloadHeader
	Volume struct {
		Level float32 `json:"level"`
	} `json:"volume"`
}

type mediaItemPayload struct {
	ContentID   string  `json:"contentId"`
	ContentType string  `json:"contentType"`
	StreamType  string  `json:"streamType"`
	Duration    float32 `json:"duration,omitempty"`
}

type loadPayload struct {
	payloadHeader
	Media       mediaItemPayload `json:"media"`
	CurrentTime int              `json:"currentTime"`
	Autoplay    bool             `json:"autoplay"`
}

type loadSpec struct {
	ContentID   string
	ContentType string
	Duration    float32
}

func newSession(ctx context.Context, target Target, load loadSpec) (*castSession, error) {
	return newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return libcast.NewConnection() })
}

func newSessionWithConnFactory(ctx context.Context, target Target, load loadSpec, makeConn connFactory) (*castSession, error) {
	s := &castSession{
		target:       target,
		conn:         makeConn(),
		waiters:      map[int]chan protocolMessage{},
		events:       make(chan sessionEvent, 8),
		stateChanged: make(chan struct{}, 1),
		terminal:     make(chan sessionEvent, 4),
		done:         make(chan struct{}),
		snapshot: sessionSnapshot{
			Host:        target.Host,
			Port:        target.portOrDefault(),
			NeverPlayed: true,
		},
	}
	if err := s.initialize(ctx, load); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *castSession) Snapshot() sessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *castSession) Events() <-chan sessionEvent { return s.events }

func (s *castSession) initialize(ctx context.Context, load loadSpec) error {
	if err := s.conn.Start(s.target.Host, s.target.portOrDefault()); err != nil {
		return err
	}
	s.mu.Lock()
	s.connStarted = true
	s.mu.Unlock()
	go s.readLoop()
	s.updateSnapshot(func(ss *sessionSnapshot) {
		ss.Connected = true
		ss.Closed = false
	})
	if err := s.sendNoWait(&connectPayload{payloadHeader{Type: "CONNECT"}}, defaultSenderID, defaultReceiverID, namespaceConn); err != nil {
		return err
	}
	if err := s.ensureDefaultMediaReceiver(ctx); err != nil {
		return err
	}
	snap := s.Snapshot()
	if snap.TransportID == "" {
		return errMissingTransport
	}
	if err := s.sendNoWait(&connectPayload{payloadHeader{Type: "CONNECT"}}, defaultSenderID, snap.TransportID, namespaceConn); err != nil {
		return err
	}
	if _, err := s.sendAndWait(ctx, &loadPayload{
		payloadHeader: payloadHeader{Type: "LOAD"},
		CurrentTime:   0,
		Autoplay:      false,
		Media: mediaItemPayload{
			ContentID:   load.ContentID,
			ContentType: load.ContentType,
			StreamType:  "BUFFERED",
			Duration:    load.Duration,
		},
	}, defaultSenderID, snap.TransportID, namespaceMedia); err != nil {
		return err
	}
	return s.waitUntilReady(ctx, load.ContentID)
}

func (s *castSession) ensureDefaultMediaReceiver(ctx context.Context) error {
	status, err := s.getReceiverStatus(ctx)
	if err != nil {
		return err
	}
	app := applicationByAppID(status, defaultMediaReceiverAppID)
	if app == nil {
		if _, err := s.sendAndWait(ctx, &launchPayload{payloadHeader: payloadHeader{Type: "LAUNCH"}, AppID: defaultMediaReceiverAppID}, defaultSenderID, defaultReceiverID, namespaceRecv); err != nil {
			return err
		}
	}
	return s.waitForReceiverApp(ctx, defaultMediaReceiverAppID)
}

func applicationByAppID(status *libcast.ReceiverStatusResponse, appID string) *libcast.Application {
	if status == nil {
		return nil
	}
	for i := range status.Status.Applications {
		app := status.Status.Applications[i]
		if app.AppId == appID {
			return &app
		}
	}
	return nil
}

func activeApplication(status *libcast.ReceiverStatusResponse) *libcast.Application {
	if app := applicationByAppID(status, defaultMediaReceiverAppID); app != nil {
		return app
	}
	if status == nil {
		return nil
	}
	for i := range status.Status.Applications {
		app := status.Status.Applications[i]
		if !app.IsIdleScreen {
			return &app
		}
	}
	if len(status.Status.Applications) > 0 {
		app := status.Status.Applications[len(status.Status.Applications)-1]
		return &app
	}
	return nil
}

func (s *castSession) waitForReceiverApp(ctx context.Context, appID string) error {
	for {
		snap := s.Snapshot()
		if snap.AppID == appID && snap.TransportID != "" {
			return nil
		}
		if _, err := s.getReceiverStatus(ctx); err != nil {
			return err
		}
		snap = s.Snapshot()
		if snap.AppID == appID && snap.TransportID != "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stateChanged:
		case ev := <-s.terminal:
			switch ev.Type {
			case sessionEventDisconnected, sessionEventClosed:
				if ev.Err != nil {
					return ev.Err
				}
				return errDisconnected
			case sessionEventLoadFailed:
				if ev.Err != nil {
					return ev.Err
				}
				return errLoadFailed
			}
		}
	}
}

func (s *castSession) getReceiverStatus(ctx context.Context) (*libcast.ReceiverStatusResponse, error) {
	msg, err := s.sendAndWait(ctx, &getStatusPayload{payloadHeader{Type: "GET_STATUS"}}, defaultSenderID, defaultReceiverID, namespaceRecv)
	if err != nil {
		return nil, err
	}
	if msg.ReceiverStatus != nil {
		return msg.ReceiverStatus, nil
	}
	return nil, fmt.Errorf("unexpected receiver status response type: %s", msg.Type)
}

func (s *castSession) waitUntilReady(ctx context.Context, contentID string) error {
	var pauseIssued bool
	for {
		snap := s.Snapshot()
		if err := readinessError(snap, contentID); err != nil {
			return err
		}
		if readyNonPlaying(snap, contentID) {
			return nil
		}
		if snap.ContentID == contentID && snap.MediaSessionID != 0 && snap.PlayerState == "PLAYING" {
			if pauseIssued {
				return errUnexpectedPlaying
			}
			pauseIssued = true
			if err := s.Pause(ctx); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stateChanged:
		case ev := <-s.terminal:
			switch ev.Type {
			case sessionEventLoadFailed:
				return fmt.Errorf("%w: %v", errLoadFailed, ev.Err)
			case sessionEventDisconnected, sessionEventClosed:
				if ev.Err != nil {
					return ev.Err
				}
				return errDisconnected
			}
		}
	}
}

func readyNonPlaying(snap sessionSnapshot, contentID string) bool {
	if snap.ContentID != contentID || snap.MediaSessionID == 0 {
		return false
	}
	switch snap.PlayerState {
	case "PAUSED":
		return true
	case "IDLE":
		return !isTerminalIdleReason(snap.IdleReason)
	default:
		return false
	}
}

func readinessError(snap sessionSnapshot, contentID string) error {
	if snap.ContentID != contentID || snap.MediaSessionID == 0 {
		return nil
	}
	if snap.PlayerState == "IDLE" && isTerminalIdleReason(snap.IdleReason) {
		return fmt.Errorf("terminal media state: %s", snap.IdleReason)
	}
	return nil
}

func isTerminalIdleReason(reason string) bool {
	switch reason {
	case "FINISHED", "ERROR", "INTERRUPTED", "CANCELLED":
		return true
	default:
		return false
	}
}

func (s *castSession) Play(ctx context.Context) error {
	snap := s.Snapshot()
	if snap.TransportID == "" || snap.MediaSessionID == 0 {
		return errMissingMedia
	}
	_, err := s.sendAndWait(ctx, &playPausePayload{payloadHeader: payloadHeader{Type: "PLAY"}, MediaSessionID: snap.MediaSessionID}, defaultSenderID, snap.TransportID, namespaceMedia)
	return err
}

func (s *castSession) Pause(ctx context.Context) error {
	snap := s.Snapshot()
	if snap.TransportID == "" || snap.MediaSessionID == 0 {
		return errMissingMedia
	}
	_, err := s.sendAndWait(ctx, &playPausePayload{payloadHeader: payloadHeader{Type: "PAUSE"}, MediaSessionID: snap.MediaSessionID}, defaultSenderID, snap.TransportID, namespaceMedia)
	return err
}

func (s *castSession) SeekTo(ctx context.Context, seconds float32, resumeState *string) error {
	snap := s.Snapshot()
	if snap.TransportID == "" || snap.MediaSessionID == 0 {
		return errMissingMedia
	}
	_, err := s.sendAndWait(ctx, &seekPayload{payloadHeader: payloadHeader{Type: "SEEK"}, MediaSessionID: snap.MediaSessionID, CurrentTime: seconds, ResumeState: resumeState}, defaultSenderID, snap.TransportID, namespaceMedia)
	return err
}

func (s *castSession) SetVolume(ctx context.Context, level float32) error {
	if level < 0 || level > 1 {
		return fmt.Errorf("cast volume out of range: %f", level)
	}
	payload := &receiverVolumePayload{payloadHeader: payloadHeader{Type: "SET_VOLUME"}}
	payload.Volume.Level = level
	_, err := s.sendAndWait(ctx, payload, defaultSenderID, defaultReceiverID, namespaceRecv)
	return err
}

// Close marks the session terminal before transport teardown. Pending waiters are
// awakened by s.done, but per-request channels are never closed because the read
// loop may still race with late responses.
func (s *castSession) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closing = true
		s.snapshot.Closed = true
		s.snapshot.Connected = false
		s.mu.Unlock()
		closeEvent := sessionEvent{Type: sessionEventClosed, Snapshot: s.Snapshot(), Err: errSessionClosed}
		s.signalTerminal(closeEvent)
		s.emitEvent(closeEvent)
		signalCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		snap := s.Snapshot()
		if snap.TransportID != "" && snap.MediaSessionID != 0 {
			_, _ = s.sendAndWait(signalCtx, &stopPayload{payloadHeader: payloadHeader{Type: "STOP"}, MediaSessionID: snap.MediaSessionID}, defaultSenderID, snap.TransportID, namespaceMedia)
		}
		close(s.done)
		s.mu.Lock()
		for id, waiter := range s.waiters {
			delete(s.waiters, id)
			select {
			case waiter <- protocolMessage{Err: errSessionClosed}:
			default:
			}
		}
		started := s.connStarted
		s.mu.Unlock()
		if started {
			err = s.conn.Close()
		}
	})
	return err
}

func (s *castSession) readLoop() {
	defer func() {
		s.eventsMu.Lock()
		s.eventsClosed = true
		close(s.events)
		s.eventsMu.Unlock()
	}()
	for msg := range s.conn.MsgChan() {
		parsed := parseProtocolMessage(msg)
		if parsed.Err != nil {
			continue
		}
		s.dispatch(parsed)
	}
	s.updateSnapshot(func(ss *sessionSnapshot) {
		ss.Connected = false
		ss.Closed = true
	})
	s.mu.RLock()
	closing := s.closing
	s.mu.RUnlock()
	if !closing {
		ev := sessionEvent{Type: sessionEventDisconnected, Snapshot: s.Snapshot(), Err: errDisconnected}
		s.signalTerminal(ev)
		s.emitEvent(ev)
	}
}

// dispatch applies snapshot updates before waking any waiter so command callers
// never observe a response whose cached state is still stale.
func (s *castSession) dispatch(msg protocolMessage) {
	switch msg.Type {
	case "RECEIVER_STATUS":
		if msg.ReceiverStatus != nil {
			s.applyReceiverStatus(msg.ReceiverStatus)
			s.signalStateChanged()
		}
	case "MEDIA_STATUS":
		if msg.MediaStatus != nil {
			s.applyMediaStatus(msg.MediaStatus)
			s.signalStateChanged()
			snap := s.Snapshot()
			if snap.MediaSessionID != 0 && snap.PlayerState == "IDLE" && snap.IdleReason == "FINISHED" {
				s.emitEvent(sessionEvent{Type: sessionEventMediaStatus, Snapshot: snap})
			}
		}
	case "LOAD_FAILED":
		ev := sessionEvent{Type: sessionEventLoadFailed, Snapshot: s.Snapshot(), Err: errLoadFailed}
		s.signalTerminal(ev)
		s.emitEvent(ev)
	}
	if msg.RequestID != 0 {
		s.mu.RLock()
		waiter := s.waiters[msg.RequestID]
		s.mu.RUnlock()
		if waiter != nil {
			select {
			case waiter <- msg:
			default:
			}
		}
	}
}

func (s *castSession) applyReceiverStatus(status *libcast.ReceiverStatusResponse) {
	app := activeApplication(status)
	s.updateSnapshot(func(ss *sessionSnapshot) {
		if app != nil {
			ss.AppID = app.AppId
			ss.AppSessionID = app.SessionId
			ss.TransportID = app.TransportId
		}
		ss.ReceiverVolume = status.Status.Volume.Level
		ss.LastUpdate = time.Now()
	})
}

func (s *castSession) applyMediaStatus(status *libcast.MediaStatusResponse) {
	if len(status.Status) == 0 {
		return
	}
	media := status.Status[len(status.Status)-1]
	s.updateSnapshot(func(ss *sessionSnapshot) {
		ss.MediaSessionID = media.MediaSessionId
		ss.PlayerState = media.PlayerState
		ss.IdleReason = media.IdleReason
		ss.CurrentTime = media.CurrentTime
		ss.Duration = media.Media.Duration
		ss.ContentID = media.Media.ContentId
		ss.MediaVolume = media.Volume.Level
		ss.LastUpdate = time.Now()
		if media.PlayerState == "PLAYING" {
			ss.NeverPlayed = false
		}
	})
}

func (s *castSession) signalStateChanged() {
	select {
	case s.stateChanged <- struct{}{}:
	default:
	}
}

func (s *castSession) signalTerminal(ev sessionEvent) {
	select {
	case s.terminal <- ev:
	default:
	}
}

func (s *castSession) updateSnapshot(fn func(*sessionSnapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.snapshot)
}

// emitEvent is reserved for runtime-critical track events only. Constructor
// readiness uses stateChanged/terminal so media-status floods cannot starve it.
func (s *castSession) emitEvent(ev sessionEvent) {
	s.eventsMu.RLock()
	defer s.eventsMu.RUnlock()
	if s.eventsClosed {
		return
	}
	select {
	case s.events <- ev:
	case <-s.done:
	}
}

func (s *castSession) sendNoWait(payload libcast.Payload, sourceID, destinationID, namespace string) error {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()
	select {
	case <-s.done:
		return errSessionClosed
	default:
	}
	s.nextRequestID++
	payload.SetRequestId(s.nextRequestID)
	return s.conn.Send(s.nextRequestID, payload, sourceID, destinationID, namespace)
}

func (s *castSession) sendAndWait(ctx context.Context, payload libcast.Payload, sourceID, destinationID, namespace string) (protocolMessage, error) {
	ctx, cancel := requestContext(ctx)
	defer cancel()

	result := make(chan protocolMessage, 1)
	var requestID int

	s.cmdMu.Lock()
	select {
	case <-s.done:
		s.cmdMu.Unlock()
		return protocolMessage{}, errSessionClosed
	default:
	}
	s.nextRequestID++
	requestID = s.nextRequestID
	payload.SetRequestId(requestID)
	s.mu.Lock()
	s.waiters[requestID] = result
	s.mu.Unlock()
	err := s.conn.Send(requestID, payload, sourceID, destinationID, namespace)
	s.cmdMu.Unlock()
	if err != nil {
		s.mu.Lock()
		delete(s.waiters, requestID)
		s.mu.Unlock()
		return protocolMessage{}, err
	}
	defer func() {
		s.mu.Lock()
		delete(s.waiters, requestID)
		s.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return protocolMessage{}, ctx.Err()
	case <-s.done:
		return protocolMessage{}, errSessionClosed
	case msg, ok := <-result:
		if !ok {
			return protocolMessage{}, errSessionClosed
		}
		if msg.Err != nil {
			return protocolMessage{}, msg.Err
		}
		if msg.Type == "LOAD_FAILED" {
			return protocolMessage{}, errLoadFailed
		}
		return msg, nil
	}
}

func requestContext(parent context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= requestTimeout {
			return context.WithDeadline(parent, deadline)
		}
	}
	return context.WithTimeout(parent, requestTimeout)
}

func parseProtocolMessage(msg *pb.CastMessage) protocolMessage {
	if msg == nil || msg.PayloadUtf8 == nil {
		return protocolMessage{Err: errors.New("empty cast message")}
	}
	var header struct {
		Type      string `json:"type"`
		RequestID int    `json:"requestId"`
	}
	if err := json.Unmarshal([]byte(*msg.PayloadUtf8), &header); err != nil {
		return protocolMessage{Err: err}
	}
	parsed := protocolMessage{Type: header.Type, RequestID: header.RequestID}
	switch header.Type {
	case "RECEIVER_STATUS":
		var status libcast.ReceiverStatusResponse
		if err := json.Unmarshal([]byte(*msg.PayloadUtf8), &status); err != nil {
			parsed.Err = err
			return parsed
		}
		parsed.ReceiverStatus = &status
	case "MEDIA_STATUS":
		var status libcast.MediaStatusResponse
		if err := json.Unmarshal([]byte(*msg.PayloadUtf8), &status); err != nil {
			parsed.Err = err
			return parsed
		}
		parsed.MediaStatus = &status
	}
	return parsed
}
