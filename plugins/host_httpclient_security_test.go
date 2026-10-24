package plugins

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/sirupsen/logrus"
)

func TestHTTPDiagnosticsKeepCredentialsOnWireOnly(t *testing.T) {
	var output bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&output)
	logger.SetLevel(logrus.TraceLevel)
	previous := log.SetDefaultLogger(logger)
	level := log.CurrentLevel()
	log.SetLevel(log.LevelTrace)
	t.Cleanup(func() { log.SetDefaultLogger(previous); log.SetLevel(level) })

	const secret = "audit-secret+/value"
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Query().Get("usertoken")
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()
	svc := newHTTPService("test-plugin", &HTTPPermission{RequiredHosts: []string{"127.0.0.1"}})
	defer svc.Close()
	response, err := svc.Send(context.Background(), host.HTTPRequest{
		Method: "GET", URL: server.URL + "/lyrics?usertoken=" + url.QueryEscape(secret) + "&%55serToken=other-secret", TimeoutMs: 1000,
	})
	if err != nil || response == nil || string(response.Body) != "ok" {
		t.Fatalf("legitimate request failed: %v", err)
	}
	if got := <-received; got != secret {
		t.Fatalf("wire credential = %q, want original value", got)
	}
	for _, value := range []string{secret, url.QueryEscape(secret), "other-secret", "usertoken="} {
		if strings.Contains(output.String(), value) {
			t.Fatal("HTTP trace exposes query credentials")
		}
	}
	if !strings.Contains(output.String(), "/lyrics") {
		t.Fatal("safe endpoint context should remain in HTTP traces")
	}
}

func TestHTTPDiagnosticsRedactFailuresAndPreserveCancellation(t *testing.T) {
	const secret = "audit-secret"
	for _, location := range []string{"http://not-allowed.invalid/lyrics?usertoken=" + secret, "/bad%zz?usertoken=" + secret} {
		t.Run(location, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", location)
				w.WriteHeader(http.StatusFound)
			}))
			defer server.Close()
			svc := newHTTPService("test-plugin", &HTTPPermission{RequiredHosts: []string{"127.0.0.1"}})
			defer svc.Close()
			_, err := svc.Send(context.Background(), host.HTTPRequest{Method: "GET", URL: server.URL + "?usertoken=" + secret})
			if err == nil || strings.Contains(err.Error(), secret) {
				t.Fatalf("expected a safe redirect failure, got %v", err)
			}
		})
	}
	svc := newHTTPService("test-plugin", &HTTPPermission{RequiredHosts: []string{"127.0.0.1"}})
	defer svc.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Send(ctx, host.HTTPRequest{Method: "GET", URL: "http://127.0.0.1:1/?usertoken=" + secret})
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), secret) {
		t.Fatalf("expected safe cancellation with intact error identity, got %v", err)
	}
}

func TestHTTPDiagnosticURLFailsClosed(t *testing.T) {
	for _, raw := range []string{
		"https://name:password@example.com/lyrics?user%74oken=a&USERtoken=b#secret",
		"https://example.com/bad%zz?usertoken=a",
	} {
		got := httpDiagnosticURL(raw)
		if strings.ContainsAny(got, "?#@") || strings.Contains(got, "password") || strings.Contains(got, "usertoken") {
			t.Fatalf("unsafe URL diagnostic %q", got)
		}
	}
}
