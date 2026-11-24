package cache

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type requestKey struct {
	Header http.Header
	URL    string
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
	key, err := cacheKey(req)
	if err != nil {
		return nil, err
	}
	ctx := req.Context()
	cached := true
	start := time.Now()
	respStr, err := c.cache.GetWithLoader(key, func(string) (string, time.Duration, error) {
		cached = false
		// Concurrent callers share this load, so one caller cancelling must not fail the others.
		resp, err := c.hc.Do(req.WithContext(context.WithoutCancel(ctx)))
		if err != nil {
			log.Trace(ctx, "CachedHTTPClient.Do", "url", req.URL, err)
			return "", 0, err
		}
		defer resp.Body.Close()
		return serializeResponse(resp), c.ttl, nil
	})
	log.Trace(ctx, "CachedHTTPClient.Do", "key", key, "cached", cached, "elapsed", time.Since(start), err)
	if err != nil {
		return nil, err
	}
	return deserializeResponse(req, respStr)
}

func cacheKey(req *http.Request) (string, error) {
	j, err := json.Marshal(requestKey{Header: req.Header, URL: req.URL.String()})
	if err != nil {
		return "", fmt.Errorf("encoding request cache key: %w", err)
	}
	return string(j), nil
}

func serializeResponse(resp *http.Response) string {
	var b = &bytes.Buffer{}
	_ = resp.Write(b)
	return b.String()
}

func deserializeResponse(req *http.Request, respStr string) (*http.Response, error) {
	r := bufio.NewReader(strings.NewReader(respStr))
	return http.ReadResponse(r, req)
}
