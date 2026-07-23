package imghttp_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/server/imghttp"
)

const testHash = "0123456789abcdef"

var lastMod = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

func found() *artwork.Image {
	return &artwork.Image{
		ReadCloser:  io.NopCloser(strings.NewReader("IMG")),
		Hash:        testHash,
		LastUpdated: lastMod,
	}
}

func placeholder() *artwork.Image {
	return &artwork.Image{ReadCloser: io.NopCloser(strings.NewReader("PH")), Placeholder: true}
}

func TestWriteImageHeaders(t *testing.T) {
	tests := []struct {
		name          string
		img           *artwork.Image
		requestedHash string
		ifNoneMatch   string
		want304       bool
		wantCache     string
		wantETag      string
		wantLastMod   bool
	}{
		{
			name:      "placeholder is never cached and carries no validators",
			img:       placeholder(),
			wantCache: "no-store",
		},
		{
			name:          "found with matching requested hash is immutable",
			img:           found(),
			requestedHash: testHash,
			wantCache:     "public, max-age=31536000, immutable",
			wantETag:      `"` + testHash + `"`,
			wantLastMod:   true,
		},
		{
			name:        "found with bare id revalidates via no-cache",
			img:         found(),
			wantCache:   "public, no-cache",
			wantETag:    `"` + testHash + `"`,
			wantLastMod: true,
		},
		{
			name:          "found with mismatched requested hash revalidates",
			img:           found(),
			requestedHash: "ffffffffffffffff",
			wantCache:     "public, no-cache",
			wantETag:      `"` + testHash + `"`,
			wantLastMod:   true,
		},
		{
			name:          "If-None-Match matching the hash yields 304",
			img:           found(),
			requestedHash: testHash,
			ifNoneMatch:   `"` + testHash + `"`,
			want304:       true,
			wantCache:     "public, max-age=31536000, immutable",
			wantETag:      `"` + testHash + `"`,
			wantLastMod:   true,
		},
		{
			name:        "weak If-None-Match matches (weak comparison)",
			img:         found(),
			ifNoneMatch: `W/"` + testHash + `"`,
			want304:     true,
			wantCache:   "public, no-cache",
			wantETag:    `"` + testHash + `"`,
			wantLastMod: true,
		},
		{
			name:        "If-None-Match with multiple values matches one",
			img:         found(),
			ifNoneMatch: `"deadbeefdeadbeef", W/"` + testHash + `", "cafecafecafecafe"`,
			want304:     true,
			wantCache:   "public, no-cache",
			wantETag:    `"` + testHash + `"`,
			wantLastMod: true,
		},
		{
			name:        "If-None-Match star matches any current representation",
			img:         found(),
			ifNoneMatch: "*",
			want304:     true,
			wantCache:   "public, no-cache",
			wantETag:    `"` + testHash + `"`,
			wantLastMod: true,
		},
		{
			name:        "non-matching If-None-Match serves body",
			img:         found(),
			ifNoneMatch: `"deadbeefdeadbeef"`,
			want304:     false,
			wantCache:   "public, no-cache",
			wantETag:    `"` + testHash + `"`,
			wantLastMod: true,
		},
		{
			name:        "placeholder ignores If-None-Match and never 304s",
			img:         placeholder(),
			ifNoneMatch: "*",
			want304:     false,
			wantCache:   "no-store",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/img", nil)
			if tc.ifNoneMatch != "" {
				r.Header.Set("If-None-Match", tc.ifNoneMatch)
			}

			got := imghttp.WriteImageHeaders(w, r, tc.img, tc.requestedHash)
			if got != tc.want304 {
				t.Fatalf("wrote304 = %v, want %v", got, tc.want304)
			}

			h := w.Header()
			if got := h.Get("Cache-Control"); got != tc.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCache)
			}
			if got := h.Get("ETag"); got != tc.wantETag {
				t.Errorf("ETag = %q, want %q", got, tc.wantETag)
			}
			if tc.img.Placeholder {
				if h.Get("ETag") != "" {
					t.Errorf("placeholder must not set ETag, got %q", h.Get("ETag"))
				}
				if h.Get("Last-Modified") != "" {
					t.Errorf("placeholder must not set Last-Modified, got %q", h.Get("Last-Modified"))
				}
			}
			if hasLM := h.Get("Last-Modified") != ""; hasLM != tc.wantLastMod {
				t.Errorf("has Last-Modified = %v, want %v", hasLM, tc.wantLastMod)
			}
			if tc.want304 {
				if w.Code != http.StatusNotModified {
					t.Errorf("status = %d, want 304", w.Code)
				}
				if w.Body.Len() != 0 {
					t.Errorf("304 must have empty body, got %q", w.Body.String())
				}
			}
		})
	}
}
