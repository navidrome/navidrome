package cast

import "time"

type Target struct {
	Host string
	Port int
	Name string
}

func (t Target) portOrDefault() int {
	if t.Port > 0 {
		return t.Port
	}
	return 8009
}

type sessionSnapshot struct {
	Host           string
	Port           int
	Connected      bool
	Closed         bool
	AppID          string
	AppSessionID   string
	TransportID    string
	MediaSessionID int
	PlayerState    string
	IdleReason     string
	CurrentTime    float32
	Duration       float32
	ContentID      string
	ReceiverVolume float32
	MediaVolume    float32
	LastUpdate     time.Time
	NeverPlayed    bool
}

type sessionEventType string

const (
	sessionEventMediaStatus    sessionEventType = "media_status"
	sessionEventReceiverStatus sessionEventType = "receiver_status"
	sessionEventLoadFailed     sessionEventType = "load_failed"
	sessionEventDisconnected   sessionEventType = "disconnected"
	sessionEventClosed         sessionEventType = "closed"
)

type sessionEvent struct {
	Type     sessionEventType
	Snapshot sessionSnapshot
	Err      error
}
