package cast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
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
	LaunchError    *launchErrorMessage
	Err            error
}

// launchErrorMessage captures only safe structured receiver-side launch
// failure fields for diagnostics.
type launchErrorMessage struct {
	Reason    string
	ErrorCode string
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
	log.Debug("Cast init: conn.Start begin", "target", s.target.Name, "host", s.target.Host, "port", s.target.portOrDefault())
	if err := s.conn.Start(s.target.Host, s.target.portOrDefault()); err != nil {
		log.Debug("Cast init: conn.Start failed", "target", s.target.Name, "host", s.target.Host, "port", s.target.portOrDefault(), "err", err)
		return fmt.Errorf("cast conn start: %w", err)
	}
	log.Debug("Cast init: conn.Start ok", "target", s.target.Name, "host", s.target.Host, "port", s.target.portOrDefault())
	s.mu.Lock()
	s.connStarted = true
	s.mu.Unlock()
	go s.readLoop()
	s.updateSnapshot(func(ss *sessionSnapshot) {
		ss.Connected = true
		ss.Closed = false
	})
	log.Debug("Cast init: receiver CONNECT begin", "target", s.target.Name)
	if err := s.sendNoWait(&connectPayload{payloadHeader{Type: "CONNECT"}}, defaultSenderID, defaultReceiverID, namespaceConn); err != nil {
		log.Debug("Cast init: receiver CONNECT failed", "target", s.target.Name, "err", err)
		return fmt.Errorf("cast receiver connect: %w", err)
	}
	log.Debug("Cast init: receiver CONNECT ok", "target", s.target.Name)
	if err := s.ensureDefaultMediaReceiver(ctx); err != nil {
		return err
	}
	snap := s.Snapshot()
	if snap.TransportID == "" {
		log.Debug("Cast init: receiver app ready but transport missing", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", false)
		return errMissingTransport
	}
	log.Debug("Cast init: media transport CONNECT begin", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true)
	if err := s.sendNoWait(&connectPayload{payloadHeader{Type: "CONNECT"}}, defaultSenderID, snap.TransportID, namespaceConn); err != nil {
		log.Debug("Cast init: media transport CONNECT failed", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true, "err", err)
		return fmt.Errorf("cast media transport connect: %w", err)
	}
	log.Debug("Cast init: media transport CONNECT ok", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true)
	log.Debug("Cast init: LOAD begin", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true)
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
		log.Debug("Cast init: LOAD failed", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true, "err", err)
		return fmt.Errorf("load media: %w", err)
	}
	log.Debug("Cast init: LOAD ack received", "target", s.target.Name, "appId", snap.AppID, "transportIdPresent", true)
	if err := s.waitUntilReady(ctx, load.ContentID); err != nil {
		return err
	}
	log.Debug("Cast init: waitUntilReady ok", "target", s.target.Name, "state", s.sanitizedStateFields())
	return nil
}

func (s *castSession) ensureDefaultMediaReceiver(ctx context.Context) error {
	status, err := s.getReceiverStatus(ctx)
	if err != nil {
		return fmt.Errorf("get receiver status: %w", err)
	}
	app := applicationByAppID(status, defaultMediaReceiverAppID)
	log.Debug("Cast init: receiver app state", "target", s.target.Name, "expectedAppId", defaultMediaReceiverAppID, "present", app != nil, "transportIdPresent", app != nil && app.TransportId != "", "applications", sanitizedApplications(status, defaultMediaReceiverAppID))
	if app == nil {
		log.Debug("Cast init: LAUNCH begin", "target", s.target.Name, "appId", defaultMediaReceiverAppID)
		msg, err := s.sendAndWait(ctx, &launchPayload{payloadHeader: payloadHeader{Type: "LAUNCH"}, AppID: defaultMediaReceiverAppID}, defaultSenderID, defaultReceiverID, namespaceRecv)
		if err != nil {
			log.Debug("Cast init: LAUNCH failed", "target", s.target.Name, "appId", defaultMediaReceiverAppID, "err", err)
			return fmt.Errorf("launch receiver app: %w", err)
		}
		if err := s.validateLaunchReply(defaultMediaReceiverAppID, msg); err != nil {
			log.Debug("Cast init: LAUNCH failed", "target", s.target.Name, "appId", defaultMediaReceiverAppID, "err", err)
			return fmt.Errorf("launch receiver app: %w", err)
		}
		log.Debug("Cast init: LAUNCH acknowledged", "target", s.target.Name, "appId", defaultMediaReceiverAppID)
	}
	log.Debug("Cast init: waitForReceiverApp begin", "target", s.target.Name, "appId", defaultMediaReceiverAppID)
	if err := s.waitForReceiverApp(ctx, defaultMediaReceiverAppID); err != nil {
		return fmt.Errorf("wait for receiver app: %w", err)
	}
	return nil
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
			log.Debug("Cast init: expected receiver app observed", "target", s.target.Name, "state", s.sanitizedStateFields())
			return nil
		}
		if _, err := s.getReceiverStatus(ctx); err != nil {
			log.Debug("Cast init: waitForReceiverApp getReceiverStatus failed", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", err)
			return fmt.Errorf("get receiver status: %w", err)
		}
		snap = s.Snapshot()
		if snap.AppID == appID && snap.TransportID != "" {
			log.Debug("Cast init: expected receiver app observed", "target", s.target.Name, "state", s.sanitizedStateFields())
			return nil
		}
		select {
		case <-ctx.Done():
			log.Debug("Cast init: waitForReceiverApp context done", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", ctx.Err())
			return ctx.Err()
		case <-s.stateChanged:
		case ev := <-s.terminal:
			switch ev.Type {
			case sessionEventDisconnected, sessionEventClosed:
				log.Debug("Cast init: waitForReceiverApp terminal", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", ev.Err)
				if ev.Err != nil {
					return ev.Err
				}
				return errDisconnected
			case sessionEventLoadFailed:
				log.Debug("Cast init: waitForReceiverApp load failed", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", ev.Err)
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
	log.Debug("Cast init: waitUntilReady begin", "target", s.target.Name, "state", s.sanitizedStateFields())
	var pauseIssued bool
	var lastLogged sessionSnapshot
	for {
		snap := s.Snapshot()
		if snap.AppID != lastLogged.AppID || (snap.TransportID != "") != (lastLogged.TransportID != "") || snap.MediaSessionID != lastLogged.MediaSessionID || snap.PlayerState != lastLogged.PlayerState || snap.IdleReason != lastLogged.IdleReason {
			log.Debug("Cast init: waitUntilReady state", "target", s.target.Name, "state", sanitizedState(snap))
			lastLogged = snap
		}
		if err := readinessError(snap, contentID); err != nil {
			log.Debug("Cast init: waitUntilReady failed", "target", s.target.Name, "state", sanitizedState(snap), "err", err)
			return fmt.Errorf("wait until ready: %w", err)
		}
		if readyNonPlaying(snap, contentID) {
			log.Debug("Cast init: waitUntilReady success", "target", s.target.Name, "state", sanitizedState(snap))
			return nil
		}
		if snap.ContentID == contentID && snap.MediaSessionID != 0 && snap.PlayerState == "PLAYING" {
			if pauseIssued {
				log.Debug("Cast init: waitUntilReady failed", "target", s.target.Name, "state", sanitizedState(snap), "err", errUnexpectedPlaying)
				return fmt.Errorf("wait until ready: %w", errUnexpectedPlaying)
			}
			pauseIssued = true
			if err := s.Pause(ctx); err != nil {
				log.Debug("Cast init: waitUntilReady pause failed", "target", s.target.Name, "state", sanitizedState(snap), "err", err)
				return fmt.Errorf("wait until ready: %w", err)
			}
		}
		select {
		case <-ctx.Done():
			log.Debug("Cast init: waitUntilReady context done", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", ctx.Err())
			return fmt.Errorf("wait until ready: %w", ctx.Err())
		case <-s.stateChanged:
		case ev := <-s.terminal:
			switch ev.Type {
			case sessionEventLoadFailed:
				err := fmt.Errorf("%w: %v", errLoadFailed, ev.Err)
				log.Debug("Cast init: waitUntilReady load failed", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", err)
				return fmt.Errorf("wait until ready: %w", err)
			case sessionEventDisconnected, sessionEventClosed:
				if ev.Err != nil {
					log.Debug("Cast init: waitUntilReady terminal", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", ev.Err)
					return fmt.Errorf("wait until ready: %w", ev.Err)
				}
				log.Debug("Cast init: waitUntilReady terminal", "target", s.target.Name, "state", s.sanitizedStateFields(), "err", errDisconnected)
				return fmt.Errorf("wait until ready: %w", errDisconnected)
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
			snap := s.Snapshot()
			if snap.MediaSessionID != 0 {
				s.emitNonBlockingEvent(sessionEvent{Type: sessionEventReceiverStatus, Snapshot: snap})
			}
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

func sanitizedState(snap sessionSnapshot) map[string]any {
	return map[string]any{
		"appId":              snap.AppID,
		"transportIdPresent": snap.TransportID != "",
		"mediaSessionId":     snap.MediaSessionID,
		"playerState":        snap.PlayerState,
		"idleReason":         snap.IdleReason,
	}
}

func sanitizedApplications(status *libcast.ReceiverStatusResponse, expectedAppID string) map[string]any {
	apps := []map[string]any{}
	anyTransport := false
	expectedPresent := false
	if status != nil {
		for _, app := range status.Status.Applications {
			transportPresent := app.TransportId != ""
			sessionPresent := app.SessionId != ""
			if transportPresent {
				anyTransport = true
			}
			if app.AppId == expectedAppID {
				expectedPresent = true
			}
			apps = append(apps, map[string]any{
				"appId":              app.AppId,
				"transportIdPresent": transportPresent,
				"sessionIdPresent":   sessionPresent,
				"isIdleScreen":       app.IsIdleScreen,
			})
		}
	}
	return map[string]any{
		"count":              len(apps),
		"expectedAppId":      expectedAppID,
		"expectedAppPresent": expectedPresent,
		"anyTransportId":     anyTransport,
		"applications":       apps,
	}
}

func (s *castSession) sanitizedStateFields() map[string]any {
	return sanitizedState(s.Snapshot())
}

func (s *castSession) validateLaunchReply(expectedAppID string, msg protocolMessage) error {
	fields := []any{
		"target", s.target.Name,
		"appId", expectedAppID,
		"replyType", msg.Type,
		"requestId", msg.RequestID,
	}
	switch msg.Type {
	case "RECEIVER_STATUS":
		fields = append(fields, "classification", "A_RECEIVER_STATUS")
		if msg.ReceiverStatus != nil {
			fields = append(fields, "applications", sanitizedApplications(msg.ReceiverStatus, expectedAppID))
		}
		log.Debug(append([]any{"Cast init: LAUNCH correlated reply"}, fields...)...)
		return nil
	case "LAUNCH_ERROR":
		fields = append(fields, "classification", "B_LAUNCH_ERROR")
		err := &launchReplyError{ReplyType: msg.Type}
		if msg.LaunchError != nil {
			fields = append(fields,
				"reason", msg.LaunchError.Reason,
				"errorCode", msg.LaunchError.ErrorCode,
			)
			err.Reason = msg.LaunchError.Reason
			err.ErrorCode = msg.LaunchError.ErrorCode
		}
		log.Debug(append([]any{"Cast init: LAUNCH correlated reply"}, fields...)...)
		return err
	default:
		fields = append(fields, "classification", "C_OTHER")
		log.Debug(append([]any{"Cast init: LAUNCH correlated reply"}, fields...)...)
		return &launchReplyError{ReplyType: msg.Type}
	}
}

type launchReplyError struct {
	ReplyType string
	Reason    string
	ErrorCode string
}

func (e *launchReplyError) Error() string {
	if e == nil {
		return "cast launch failed"
	}
	if e.ReplyType == "LAUNCH_ERROR" {
		switch {
		case e.Reason != "" && e.ErrorCode != "":
			return fmt.Sprintf("cast launch failed: replyType=%s reason=%s errorCode=%s", e.ReplyType, e.Reason, e.ErrorCode)
		case e.Reason != "":
			return fmt.Sprintf("cast launch failed: replyType=%s reason=%s", e.ReplyType, e.Reason)
		case e.ErrorCode != "":
			return fmt.Sprintf("cast launch failed: replyType=%s errorCode=%s", e.ReplyType, e.ErrorCode)
		}
	}
	if e.ReplyType != "" {
		return fmt.Sprintf("cast launch failed: unexpected correlated reply type: %s", e.ReplyType)
	}
	return "cast launch failed"
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

// emitNonBlockingEvent is best-effort for state sync updates. Dropping an
// intermediate receiver-status event is acceptable because the latest snapshot
// remains available without risking constructor/read-loop stalls.
func (s *castSession) emitNonBlockingEvent(ev sessionEvent) {
	s.eventsMu.RLock()
	defer s.eventsMu.RUnlock()
	if s.eventsClosed {
		return
	}
	select {
	case s.events <- ev:
	case <-s.done:
	default:
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
	case "LAUNCH_ERROR":
		var launchErr struct {
			Reason    string `json:"reason"`
			ErrorCode string `json:"errorCode"`
		}
		if err := json.Unmarshal([]byte(*msg.PayloadUtf8), &launchErr); err != nil {
			parsed.Err = err
			return parsed
		}
		parsed.LaunchError = &launchErrorMessage{Reason: launchErr.Reason, ErrorCode: launchErr.ErrorCode}
	}
	return parsed
}
