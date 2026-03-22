// Package radiobrowser queries the community Radio Browser API (https://api.radio-browser.info).
package radiobrowser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/navidrome/navidrome/consts"
)

const (
	defaultLimit = 30
	maxLimit     = 100
	minQueryLen  = 2
	maxQueryLen  = 200
)

// Station is a minimal subset returned to the Navidrome UI.
type Station struct {
	StationUUID string `json:"stationuuid"`
	Name        string `json:"name"`
	StreamURL   string `json:"streamUrl"`
	HomePageURL string `json:"homePageUrl"`
}

type apiStation struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	URLResolved string `json:"url_resolved"`
	Homepage    string `json:"homepage"`
	StationUUID string `json:"stationuuid"`
}

// Sentinel errors for query validation. Use errors.Is to detect them.
var (
	ErrQueryTooShort = errors.New("query too short")
	ErrQueryTooLong  = errors.New("query too long")
)

var fallbackAPIHosts = []string{
	"de1.api.radio-browser.info",
	"nl1.api.radio-browser.info",
	"at1.api.radio-browser.info",
	"fi1.api.radio-browser.info",
}

// APIHosts returns hostnames of Radio Browser API servers (DNS with static fallback).
func APIHosts() []string {
	ips, err := net.LookupIP("all.api.radio-browser.info")
	if err != nil || len(ips) == 0 {
		return append([]string(nil), fallbackAPIHosts...)
	}
	seen := map[string]struct{}{}
	var hosts []string
	for _, ip := range ips {
		names, err := net.LookupAddr(ip.String())
		if err != nil {
			continue
		}
		for _, n := range names {
			h := strings.TrimSuffix(n, ".")
			if h == "" {
				continue
			}
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		return append([]string(nil), fallbackAPIHosts...)
	}
	return hosts
}

func shuffleHosts(hosts []string) []string {
	out := append([]string(nil), hosts...)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// Search queries station search on the Radio Browser network.
func Search(ctx context.Context, rawQuery string, limit int) ([]Station, error) {
	q := strings.TrimSpace(rawQuery)
	if len(q) < minQueryLen {
		return nil, fmt.Errorf("query too short (min %d characters): %w", minQueryLen, ErrQueryTooShort)
	}
	if len(q) > maxQueryLen {
		return nil, fmt.Errorf("query too long (max %d characters): %w", maxQueryLen, ErrQueryTooLong)
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	path := "/json/stations/search?name=" + url.QueryEscape(q) +
		"&limit=" + fmt.Sprintf("%d", limit) + "&order=votes&reverse=true"

	body, err := get(ctx, path)
	if err != nil {
		return nil, err
	}

	var raw []apiStation
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse stations: %w", err)
	}

	return normalizeStations(raw), nil
}

func normalizeStations(raw []apiStation) []Station {
	out := make([]Station, 0, len(raw))
	for _, s := range raw {
		stream := strings.TrimSpace(s.URLResolved)
		if stream == "" {
			stream = strings.TrimSpace(s.URL)
		}
		if stream == "" {
			continue
		}
		out = append(out, Station{
			StationUUID: s.StationUUID,
			Name:        strings.TrimSpace(s.Name),
			StreamURL:   stream,
			HomePageURL: strings.TrimSpace(s.Homepage),
		})
	}
	return out
}

// NotifyClick reports a station stream URL click to the Radio Browser API (best-effort).
func NotifyClick(ctx context.Context, streamURL string) {
	u := strings.TrimSpace(streamURL)
	if u == "" {
		return
	}
	path := "/json/url?url=" + url.QueryEscape(u)
	_, _ = get(ctx, path)
}

func get(ctx context.Context, path string) ([]byte, error) {
	hosts := shuffleHosts(APIHosts())
	client := &http.Client{Timeout: 20 * time.Second}
	var lastErr error
	for _, host := range hosts {
		reqURL := "https://" + host + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", consts.HTTPUserAgent)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("radio-browser %s: status %d", host, resp.StatusCode)
			continue
		}
		return body, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no radio-browser server responded")
	}
	return nil, lastErr
}
