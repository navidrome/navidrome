package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/sirupsen/logrus"
)

// LogEntry represents a log entry in the format sent to the UI
type LogEntry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type Router struct {
	http.Handler
	ds model.DataStore
}

func NewRouter(ds model.DataStore) *Router {
	r := &Router{ds: ds}
	r.Handler = r.routes()
	return r
}

func (s *Router) routes() http.Handler {
	r := chi.NewRouter()

	// Admin-only routes
	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(s.ds))
		r.Use(server.JWTRefresher)
		r.Use(server.RequireAdmin)

		r.Get("/logs/stream", s.streamLogs)
	})

	return r
}

func (s *Router) streamLogs(w http.ResponseWriter, r *http.Request) {
	// Check if the user is an admin
	user, _ := request.UserFrom(r.Context())
	if !user.IsAdmin {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering in Nginx

	// Create a channel for receiving logs
	logCh := make(chan *logrus.Entry, 100)
	log.RegisterLogListener(logCh)
	defer log.UnregisterLogListener(logCh)

	// Send initial log buffer snapshot
	snapshot := log.GetLogBuffer().GetAll()
	for _, entry := range snapshot {
		if err := sendLogEntry(w, entry); err != nil {
			log.Error(r.Context(), "Error sending log snapshot", err)
			return
		}
	}
	
	// Flush the response writer
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Keep the connection open and stream new logs
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// Send a keep-alive message
			_, err := w.Write([]byte(": keepalive\n\n"))
			if err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case entry := <-logCh:
			if err := sendLogEntry(w, entry); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func sendLogEntry(w http.ResponseWriter, entry *logrus.Entry) error {
	// Convert logrus.Entry to LogEntry
	le := LogEntry{
		Time:    entry.Time,
		Level:   entry.Level.String(),
		Message: entry.Message,
		Data:    make(map[string]interface{}),
	}

	// Copy fields, filtering out sensitive data
	for k, v := range entry.Data {
		if k == "requestId" || k == " source" {
			continue // Skip internal fields
		}
		le.Data[k] = v
	}

	// Serialize to JSON
	data, err := json.Marshal(le)
	if err != nil {
		return err
	}

	// Write the event
	_, err = w.Write([]byte("data: " + string(data) + "\n\n"))
	return err
}