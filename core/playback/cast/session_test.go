package cast

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libcast "github.com/vishen/go-chromecast/cast"
	pb "github.com/vishen/go-chromecast/cast/proto"
)

type sentMessage struct {
	RequestID     int
	SourceID      string
	DestinationID string
	Namespace     string
	Payload       map[string]any
}

type fakeConn struct {
	mu                     sync.Mutex
	msgCh                  chan *pb.CastMessage
	sent                   []sentMessage
	onSend                 func(sentMessage)
	started                []string
	debugCalls             int
	closed                 bool
	current                int32
	maxCurrent             int32
	startErr               error
	unsafeCloseBeforeStart bool
}

func newFakeConn() *fakeConn {
	return &fakeConn{msgCh: make(chan *pb.CastMessage, 32)}
}

func (f *fakeConn) Start(addr string, port int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, fmt.Sprintf("%s:%d", addr, port))
	return f.startErr
}

func (f *fakeConn) MsgChan() chan *pb.CastMessage { return f.msgCh }

func (f *fakeConn) Close() error {
	f.mu.Lock()
	alreadyClosed := f.closed
	started := len(f.started) > 0 && f.startErr == nil
	f.closed = true
	f.mu.Unlock()
	if f.unsafeCloseBeforeStart && !started {
		panic("unsafe close before start")
	}
	if !alreadyClosed {
		close(f.msgCh)
	}
	return nil
}

func (f *fakeConn) SetDebug(debug bool)         { f.debugCalls++ }
func (f *fakeConn) LocalAddr() (string, error)  { return "127.0.0.1", nil }
func (f *fakeConn) RemoteAddr() (string, error) { return "127.0.0.2", nil }
func (f *fakeConn) RemotePort() (string, error) { return "8009", nil }

func (f *fakeConn) Send(requestID int, payload libcast.Payload, sourceID, destinationID, namespace string) error {
	current := atomic.AddInt32(&f.current, 1)
	for {
		max := atomic.LoadInt32(&f.maxCurrent)
		if current <= max || atomic.CompareAndSwapInt32(&f.maxCurrent, max, current) {
			break
		}
	}
	defer atomic.AddInt32(&f.current, -1)

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var body map[string]any
	Expect(json.Unmarshal(b, &body)).To(Succeed())
	msg := sentMessage{RequestID: requestID, SourceID: sourceID, DestinationID: destinationID, Namespace: namespace, Payload: body}
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.mu.Unlock()
	if f.onSend != nil {
		f.onSend(msg)
	}
	return nil
}

func (f *fakeConn) emit(payload any, requestID int, sourceID, destinationID, namespace string) {
	b, err := json.Marshal(payload)
	Expect(err).ToNot(HaveOccurred())
	utf8 := string(b)
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if closed {
		return
	}
	f.msgCh <- &pb.CastMessage{PayloadUtf8: &utf8, SourceId: &sourceID, DestinationId: &destinationID, Namespace: &namespace}
}

func (f *fakeConn) sentMessages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentMessage, len(f.sent))
	copy(out, f.sent)
	return out
}

func receiverStatus(requestID int, appID, transportID string) any {
	return receiverStatusWithApps(requestID, []map[string]any{{
		"appId":       appID,
		"sessionId":   "session-1",
		"transportId": transportID,
	}})
}

func receiverStatusWithApps(requestID int, apps []map[string]any) any {
	return map[string]any{
		"type":      "RECEIVER_STATUS",
		"requestId": requestID,
		"status": map[string]any{
			"volume":       map[string]any{"level": 0.5},
			"applications": apps,
		},
	}
}

func mediaStatus(requestID int, mediaSessionID int, playerState, idleReason, contentID string, currentTime float32) any {
	return map[string]any{
		"type":      "MEDIA_STATUS",
		"requestId": requestID,
		"status": []map[string]any{{
			"mediaSessionId": mediaSessionID,
			"playerState":    playerState,
			"idleReason":     idleReason,
			"currentTime":    currentTime,
			"volume":         map[string]any{"level": 0.5},
			"media": map[string]any{
				"contentId":   contentID,
				"contentType": "audio/mpeg",
				"streamType":  "BUFFERED",
				"duration":    120,
			},
		}},
	}
}

var _ = Describe("castSession", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		target Target
		load   loadSpec
		conn   *fakeConn
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		target = Target{Host: "127.0.0.1", Port: 8009, Name: "Living Room"}
		load = loadSpec{ContentID: "https://example.com/share/playback/token", ContentType: "audio/mpeg", Duration: 120}
		conn = newFakeConn()
	})

	AfterEach(func() { cancel() })

	It("connects directly, uses the default media receiver, and loads autoplay=false", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })

		msgs := conn.sentMessages()
		Expect(msgs).To(HaveLen(4))
		Expect(msgs[0].Payload["type"]).To(Equal("CONNECT"))
		Expect(msgs[0].Namespace).To(Equal(namespaceConn))
		Expect(msgs[1].Payload["type"]).To(Equal("GET_STATUS"))
		Expect(msgs[2].Payload["type"]).To(Equal("CONNECT"))
		Expect(msgs[2].DestinationID).To(Equal("transport-1"))
		Expect(msgs[3].Payload["type"]).To(Equal("LOAD"))
		Expect(msgs[3].Payload["autoplay"]).To(BeFalse())
		Expect(msgs[3].Payload["currentTime"]).To(BeNumerically("==", 0))
		media := msgs[3].Payload["media"].(map[string]any)
		Expect(media["contentId"]).To(Equal(load.ContentID))
		Expect(media["contentType"]).To(Equal(load.ContentType))
		Expect(media["streamType"]).To(Equal("BUFFERED"))
		Expect(conn.debugCalls).To(Equal(0))
		Expect(sess.Snapshot().PlayerState).To(Equal("PAUSED"))
	})

	It("launches the default media receiver when needed", func() {
		var getStatusCalls int
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				getStatusCalls++
				if getStatusCalls == 1 {
					conn.emit(receiverStatus(msg.RequestID, "OTHER_APP", "transport-other"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
				} else {
					conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-2"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
				}
			case "LAUNCH":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-2"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 88, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-2", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })

		msgs := conn.sentMessages()
		Expect(msgs[2].Payload["type"]).To(Equal("LAUNCH"))
		Expect(msgs[2].Payload["appId"]).To(Equal(defaultMediaReceiverAppID))
	})

	It("fails on LOAD_FAILED", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(map[string]any{"type": "LOAD_FAILED", "requestId": msg.RequestID}, msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		_, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).To(MatchError(ContainSubstring("cast load failed")))
	})

	It("fails if the receiver disconnects during construction", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "BUFFERING", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
				Expect(conn.Close()).To(Succeed())
			}
		}

		_, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).To(MatchError(ContainSubstring("cast connection disconnected")))
	})

	It("pauses unexpected playback before succeeding", func() {
		var pauseSeen atomic.Bool
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PLAYING", "", load.ContentID, 1), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "PAUSE":
				pauseSeen.Store(true)
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 1), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })
		Expect(pauseSeen.Load()).To(BeTrue())
		Expect(sess.Snapshot().PlayerState).To(Equal("PAUSED"))
	})

	It("serializes concurrent commands", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "PLAY", "PAUSE":
				time.Sleep(50 * time.Millisecond)
				state := "PLAYING"
				if msg.Payload["type"] == "PAUSE" {
					state = "PAUSED"
				}
				conn.emit(mediaStatus(msg.RequestID, 77, state, "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer GinkgoRecover(); defer wg.Done(); Expect(sess.Play(ctx)).To(Succeed()) }()
		go func() { defer GinkgoRecover(); defer wg.Done(); Expect(sess.Pause(ctx)).To(Succeed()) }()
		wg.Wait()
		Expect(atomic.LoadInt32(&conn.maxCurrent)).To(Equal(int32(1)))
	})

	It("sends seek payloads with and without resumeState as requested", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "SEEK":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 42), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })

		resume := "PLAYBACK_START"
		Expect(sess.SeekTo(ctx, 42, &resume)).To(Succeed())
		Expect(sess.SeekTo(ctx, 7, nil)).To(Succeed())

		msgs := conn.sentMessages()
		var seeks []sentMessage
		for _, msg := range msgs {
			if msg.Payload["type"] == "SEEK" {
				seeks = append(seeks, msg)
			}
		}
		Expect(seeks).To(HaveLen(2))
		Expect(seeks[0].Payload["resumeState"]).To(Equal("PLAYBACK_START"))
		_, hasResume := seeks[1].Payload["resumeState"]
		Expect(hasResume).To(BeFalse())
	})

	It("waits for delayed default media receiver transport after launch", func() {
		var getStatusCalls int
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				getStatusCalls++
				switch getStatusCalls {
				case 1:
					conn.emit(receiverStatus(msg.RequestID, "OTHER_APP", "transport-other"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
				case 2:
					conn.emit(receiverStatusWithApps(msg.RequestID, []map[string]any{{"appId": defaultMediaReceiverAppID, "sessionId": "session-1", "transportId": ""}}), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
				default:
					conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-2"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
				}
			case "LAUNCH":
				conn.emit(receiverStatusWithApps(msg.RequestID, []map[string]any{{"appId": defaultMediaReceiverAppID, "sessionId": "session-1", "transportId": ""}}), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-2", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })
		Expect(sess.Snapshot().TransportID).To(Equal("transport-2"))
		Expect(getStatusCalls).To(BeNumerically(">=", 3))
	})

	It("prefers the default media receiver when multiple applications are present", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatusWithApps(msg.RequestID, []map[string]any{
					{"appId": "OTHER_APP", "sessionId": "session-other", "transportId": "transport-other"},
					{"appId": defaultMediaReceiverAppID, "sessionId": "session-1", "transportId": "transport-dmr"},
				}), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-dmr", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })
		Expect(sess.Snapshot().TransportID).To(Equal("transport-dmr"))
	})

	It("delivers readiness-critical media events even under heavy status traffic", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				for i := 0; i < 64; i++ {
					conn.emit(mediaStatus(msg.RequestID, 77, "BUFFERING", "", load.ContentID, float32(i)), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
				}
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 64), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })
		Expect(sess.Snapshot().PlayerState).To(Equal("PAUSED"))
	})

	It("applies response state before waking waiters", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PLAYING", "", load.ContentID, 1), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "PAUSE":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 1), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = sess.Close() })
		Expect(sess.Snapshot().PlayerState).To(Equal("PAUSED"))
	})

	It("tolerates late responses after timeout and close", func() {
		late := make(chan sentMessage, 1)
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "PLAY":
				late <- msg
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		playCtx, playCancel := context.WithTimeout(ctx, 20*time.Millisecond)
		defer playCancel()
		Expect(sess.Play(playCtx)).To(HaveOccurred())
		Expect(sess.Close()).To(Succeed())
		msg := <-late
		conn.emit(mediaStatus(msg.RequestID, 77, "PLAYING", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
	})

	It("survives close racing with late responses", func() {
		late := make(chan sentMessage, 1)
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "PLAY":
				late <- msg
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		playDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			_ = sess.Play(ctx)
			close(playDone)
		}()
		msg := <-late
		Expect(sess.Close()).To(Succeed())
		conn.emit(mediaStatus(msg.RequestID, 77, "PLAYING", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
		Eventually(playDone).Should(BeClosed())
	})

	It("does not call unsafe close when start fails", func() {
		conn.startErr = fmt.Errorf("boom")
		conn.unsafeCloseBeforeStart = true
		_, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).To(MatchError("boom"))
	})

	It("best-effort stops media and closes cleanly", func() {
		conn.onSend = func(msg sentMessage) {
			switch msg.Payload["type"] {
			case "GET_STATUS":
				conn.emit(receiverStatus(msg.RequestID, defaultMediaReceiverAppID, "transport-1"), msg.RequestID, defaultReceiverID, defaultSenderID, namespaceRecv)
			case "LOAD":
				conn.emit(mediaStatus(msg.RequestID, 77, "PAUSED", "", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			case "STOP":
				conn.emit(mediaStatus(msg.RequestID, 77, "IDLE", "INTERRUPTED", load.ContentID, 0), msg.RequestID, "transport-1", defaultSenderID, namespaceMedia)
			}
		}

		sess, err := newSessionWithConnFactory(ctx, target, load, func() libcast.Conn { return conn })
		Expect(err).ToNot(HaveOccurred())
		Expect(sess.Close()).To(Succeed())

		msgs := conn.sentMessages()
		Expect(msgs[len(msgs)-1].Payload["type"]).To(Equal("STOP"))
	})
})
