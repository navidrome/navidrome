package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxSourceBytes = 65536

var tagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*))?$`)

type releaseRecord struct {
	ID          int64  `json:"id"`
	Tag         string `json:"tag_name"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	UpdatedAt   string `json:"updated_at"`
}

type source struct {
	ID          int64  `json:"id"`
	SourceID    string `json:"source_id"`
	Tag         string `json:"tag"`
	URL         string `json:"url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	UpdatedAt   string `json:"updated_at"`
	Prerelease  bool   `json:"prerelease"`
}

func version(tag string) ([3]uint64, error) {
	var v [3]uint64
	matches := tagPattern.FindStringSubmatch(tag)
	if len(tag) > 80 || matches == nil {
		return v, errors.New("versions must look like v0.64.2")
	}
	for i := range v {
		n, err := strconv.ParseUint(matches[i+1], 10, 32)
		if err != nil {
			return v, errors.New("version component exceeds limit")
		}
		v[i] = n
	}
	if matches[4] != "" {
		for _, identifier := range strings.Split(matches[4], ".") {
			if numericIdentifier(identifier) && len(identifier) > 1 && identifier[0] == '0' {
				return v, errors.New("numeric prerelease identifiers must not have leading zeros")
			}
		}
	}
	return v, nil
}

func normalizeTag(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	_, err := version(tag)
	return tag, err
}

func compareVersion(a, b string) int {
	va, _ := version(a)
	vb, _ := version(b)
	for i := range va {
		if va[i] < vb[i] {
			return -1
		}
		if va[i] > vb[i] {
			return 1
		}
	}
	_, pa, _ := strings.Cut(a, "-")
	_, pb, _ := strings.Cut(b, "-")
	return comparePrerelease(pa, pb)
}

func numericIdentifier(s string) bool { return strings.Trim(s, "0123456789") == "" }

func compareIdentifiers(a, b string) int {
	an, bn := numericIdentifier(a), numericIdentifier(b)
	if an != bn {
		if an {
			return -1
		}
		return 1
	}
	// Numeric identifiers are valid without leading zeros; length then lexical
	// comparison handles arbitrarily large identifiers without integer overflow.
	if an && len(a) != len(b) {
		return len(a) - len(b)
	}
	return strings.Compare(a, b)
}

func comparePrerelease(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1 // A stable release follows every prerelease of the same version.
	}
	if b == "" {
		return -1
	}
	ai, bi := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(ai), len(bi)) {
		if order := compareIdentifiers(ai[i], bi[i]); order != 0 {
			return order
		}
	}
	return len(ai) - len(bi)
}

func normalizeRelease(r releaseRecord, allowPrerelease bool) (source, error) {
	if _, err := version(r.Tag); err != nil {
		return source{}, err
	}
	if r.ID <= 0 || r.Draft || r.PublishedAt == "" {
		return source{}, errors.New("only published releases with valid IDs are eligible")
	}
	if r.Prerelease && !allowPrerelease {
		return source{}, errors.New("published prereleases require explicit opt-in")
	}
	if strings.TrimSpace(r.Body) == "" {
		return source{}, errors.New("release notes are empty; add notes before generating")
	}
	if len(r.Body) > maxSourceBytes {
		return source{}, errors.New("release notes exceed the byte limit; never truncate warnings")
	}
	return source{ID: r.ID, SourceID: strconv.FormatInt(r.ID, 10), Tag: r.Tag,
		URL:  "https://github.com/" + repository + "/releases/tag/" + url.PathEscape(r.Tag),
		Body: strings.ReplaceAll(r.Body, "\r\n", "\n"), PublishedAt: r.PublishedAt, UpdatedAt: r.UpdatedAt, Prerelease: r.Prerelease}, nil
}

func (e *engine) request(ctx context.Context, endpoint, token string, payload any, limit int64) ([]byte, string, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, "", errors.New("cannot encode request")
		}
		body = bytes.NewReader(data)
	}
	method := http.MethodGet
	if payload != nil {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, "", errors.New("invalid request")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := e.client.Do(req)
	if err != nil {
		return nil, "", errors.New("network request failed; requests are never retried")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("HTTP %d; requests are never retried", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, "", errors.New("cannot read response; requests are never retried")
	}
	if int64(len(data)) > limit {
		return nil, "", errors.New("response exceeds byte limit")
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	return data, contentType, nil
}

func (e *engine) github(ctx context.Context, path string, target any) error {
	data, _, err := e.request(ctx, "https://api.github.com/repos/"+repository+path, e.githubToken, nil, 8<<20)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return errors.New("invalid GitHub response")
	}
	return nil
}

func (e *engine) fetchTag(ctx context.Context, tag string, allow bool) (source, error) {
	var record releaseRecord
	if err := e.github(ctx, "/releases/tags/"+url.PathEscape(tag), &record); err != nil {
		return source{}, err
	}
	if record.Tag != tag {
		return source{}, errors.New("release tag identity changed")
	}
	return normalizeRelease(record, allow)
}

func (e *engine) resolveLocal(ctx context.Context, o options) ([]source, error) {
	if o.from == "" {
		return nil, errors.New("--from is required; add --to for an inclusive range")
	}
	if o.to == "" {
		tag, err := normalizeTag(o.from)
		if err != nil {
			return nil, err
		}
		s, err := e.fetchTag(ctx, tag, o.includePrereleases)
		if err != nil {
			return nil, err
		}
		return []source{s}, nil
	}
	return e.resolveRange(ctx, o)
}

func (e *engine) resolveRange(ctx context.Context, o options) ([]source, error) {
	if o.from == "" || o.to == "" {
		return nil, errors.New("range requires both --from and --to")
	}
	from, err := normalizeTag(o.from)
	if err != nil {
		return nil, err
	}
	to, err := normalizeTag(o.to)
	if err != nil {
		return nil, err
	}
	if strings.Contains(from, "-") || strings.Contains(to, "-") || compareVersion(from, to) > 0 {
		return nil, errors.New("range endpoints must be ordered stable versions; select a prerelease with --from alone")
	}
	var selected []source
	complete := false
	// Never assume API publication order is version order. Scan a bounded list;
	// discard unpublished/draft bodies and never send them to OpenAI or artifacts.
	for page := 1; page <= 10; page++ {
		var records []releaseRecord
		if err := e.github(ctx, fmt.Sprintf("/releases?per_page=100&page=%d", page), &records); err != nil {
			return nil, err
		}
		for _, r := range records {
			if r.Draft || r.PublishedAt == "" {
				continue
			}
			if _, err := version(r.Tag); err != nil {
				continue
			}
			if r.Prerelease && !o.includePrereleases {
				continue
			}
			if compareVersion(r.Tag, from) < 0 || compareVersion(r.Tag, to) > 0 {
				continue
			}
			s, err := normalizeRelease(r, o.includePrereleases)
			if err != nil {
				return nil, err
			}
			selected = append(selected, s)
			if len(selected) > 3 {
				return nil, errors.New("range selects more than three releases; narrow the range")
			}
		}
		if len(records) < 100 {
			complete = true
			break
		}
	}
	if !complete {
		return nil, errors.New("release listing exceeds 1000-record limit; select a single release with --from")
	}
	seenFrom, seenTo := false, false
	for _, s := range selected {
		seenFrom = seenFrom || s.Tag == from
		seenTo = seenTo || s.Tag == to
	}
	if !seenFrom || !seenTo {
		return nil, errors.New("range endpoints must both be published releases")
	}
	sort.Slice(selected, func(i, j int) bool { return compareVersion(selected[i].Tag, selected[j].Tag) < 0 })
	return selected, nil
}

func (e *engine) recheck(ctx context.Context, m manifest, sources []source) error {
	var current []source
	for _, s := range sources {
		var r releaseRecord
		if err := e.github(ctx, fmt.Sprintf("/releases/%d", s.ID), &r); err != nil {
			return err
		}
		n, err := normalizeRelease(r, m.IncludePrereleases)
		if err != nil {
			return err
		}
		current = append(current, n)
	}
	if hashJSON(current) != m.SourceHash {
		return errors.New("published notes changed or were withdrawn; review a fresh attempt")
	}
	return nil
}
