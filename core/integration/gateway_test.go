package integration

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDestinationFromHost(t *testing.T) {
	cases := map[string]Destination{
		"ws.audioscrobbler.com": DestLastFM,
		"www.last.fm":           DestLastFM,
		"libre.fm":              DestLibreFM,
		"api.listenbrainz.org":  DestListenBrainz,
		"api.deezer.com":        DestDeezer,
		"www.navidrome.org":     DestInsights,
		"example.com":           DestUnknown,
	}
	for host, want := range cases {
		if got := DestinationFromHost(host); got != want {
			t.Fatalf("host %s: got %s want %s", host, got, want)
		}
	}
}

func TestCircuitBreakerOpensAndRecovers(t *testing.T) {
	b := &circuitBreaker{}
	for i := 0; i < breakerFailThreshold; i++ {
		if !b.allow() {
			t.Fatalf("allowed before open at %d", i)
		}
		b.failure()
	}
	if b.allow() {
		t.Fatal("should be open")
	}
	b.openedAt = time.Now().Add(-breakerOpenFor - time.Second)
	if !b.allow() {
		t.Fatal("should half-open after timeout")
	}
	b.success()
	if !b.allow() {
		t.Fatal("should be closed after success")
	}
}

func TestSignAudioscrobbler(t *testing.T) {
	vector := loadSignVector(t)
	sig := signAudioscrobbler(vector.Params, vector.Secret)
	if sig != vector.Expected {
		t.Fatalf("sig = %s want %s", sig, vector.Expected)
	}
}

func TestWorkerResponseError(t *testing.T) {
	err := workerResponseError("circuit open for lastfm")
	if !errors.Is(err, errCircuitOpen) {
		t.Fatalf("expected errCircuitOpen, got %v", err)
	}
	if !isWorkerCircuitOpen(err) {
		t.Fatal("isWorkerCircuitOpen should be true")
	}

	other := workerResponseError("connection refused")
	if isWorkerCircuitOpen(other) {
		t.Fatal("isWorkerCircuitOpen should be false for transport errors")
	}
}

func TestGrpcListenAddrPrefersEnv(t *testing.T) {
	t.Setenv("ND_INTEGRATIONGRPCLISTEN", "unix:/tmp/custom.sock")
	got := grpcListenAddr()
	if got != "unix:/tmp/custom.sock" {
		t.Fatalf("got %q want env override", got)
	}
}

func TestGatewayRejectsUnknownHostWhenWorkerRequired(t *testing.T) {
	g := &Gateway{fallback: http.DefaultTransport, workerExpected: true}
	req, err := http.NewRequest(http.MethodGet, "https://example.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := g.roundTripDest(req, DestUnknown)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "unknown outbound host") {
		t.Fatalf("expected unknown host rejection, got %v", err)
	}
}

func TestGatewayFallbackRoundTrip(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(upstream.Close)

	g := &Gateway{fallback: http.DefaultTransport}
	req, err := http.NewRequest(http.MethodGet, upstream.URL+"/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := g.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
