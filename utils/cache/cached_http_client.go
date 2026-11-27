package cache

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

const cacheSizeLimit = 100

type HTTPClient struct {
	cache SimpleCache[string, string]
	hc    httpDoer
	ttl   time.Duration
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewHTTPClient(wrapped httpDoer, ttl time.Duration) *HTTPClient {
	c := &HTTPClient{hc: wrapped, ttl: ttl}
	c.cache = NewSimpleCache[string, string](Options{
		SizeLimit:  cacheSizeLimit,
		DefaultTTL: ttl,
	})
	return c
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Only GET responses are cached. Replaying a stored reply to a POST (a scrobble, a now-playing
	// update) would hand a retry the previous failure without reaching the service.
	if req.Method != http.MethodGet {
		return c.hc.Do(req)
	}
	ctx := req.Context()
	cached := true
	start := time.Now()
	respStr, err := c.cache.GetWithLoader(cacheKey(req), func(string) (string, time.Duration, error) {
		cached = false
		// Concurrent callers share this load, so one caller cancelling must not fail the others.
		resp, err := c.hc.Do(req.WithContext(context.WithoutCancel(ctx)))
		if err != nil {
			return "", 0, err
		}
		defer resp.Body.Close()
		var b bytes.Buffer
		_ = resp.Write(&b)
		return b.String(), c.ttl, nil
	})
	log.Trace(ctx, "CachedHTTPClient.Do", "url", req.URL, "cached", cached, "elapsed", time.Since(start), err)
	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(strings.NewReader(respStr)), req)
}

func cacheKey(req *http.Request) string {
	var b strings.Builder
	b.WriteString(req.URL.String())
	b.WriteByte('\n')
	_ = req.Header.Write(&b)
	return b.String()
}
