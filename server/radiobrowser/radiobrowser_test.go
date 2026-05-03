package radiobrowser

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNormalizeStations(t *testing.T) {
	raw := []apiStation{
		{Name: "A", URLResolved: "https://a.example/stream", Homepage: "https://a.example", StationUUID: "1"},
		{Name: "B", URL: "http://b-only", StationUUID: "2"},
		{Name: "skip", StationUUID: "3"},
	}
	got := normalizeStations(raw)
	if len(got) != 2 {
		t.Fatalf("len %d, want 2", len(got))
	}
	if got[0].StreamURL != "https://a.example/stream" || got[0].HomePageURL != "https://a.example" {
		t.Fatalf("first: %+v", got[0])
	}
	if got[1].StreamURL != "http://b-only" {
		t.Fatalf("second stream: %q", got[1].StreamURL)
	}
}

func TestSearchSentinelErrors(t *testing.T) {
	ctx := context.Background()
	_, err := Search(ctx, "x", 10)
	if !errors.Is(err, ErrQueryTooShort) {
		t.Fatalf("short query: want ErrQueryTooShort, got %v", err)
	}
	long := strings.Repeat("a", maxQueryLen+1)
	_, err = Search(ctx, long, 10)
	if !errors.Is(err, ErrQueryTooLong) {
		t.Fatalf("long query: want ErrQueryTooLong, got %v", err)
	}
}
