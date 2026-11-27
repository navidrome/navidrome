package storage

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/utils/slice"
)

const LocalSchemaID = "file"

type constructor func(url.URL) Storage

var (
	registry = map[string]constructor{}
	lock     sync.RWMutex
)

func Register(schema string, c constructor) {
	lock.Lock()
	defer lock.Unlock()
	registry[schema] = c
}

// LocalPathToURL converts a bare OS filesystem path into an absolute file:// URL,
// applying the same slash-normalisation and per-component escaping the scanner
// relies on. It is the single source of truth for how a local path becomes a
// storage URL, shared by For and by storage tests.
func LocalPathToURL(osPath string) (url.URL, error) {
	abs, _ := filepath.Abs(osPath)
	abs = filepath.ToSlash(abs)

	// Properly escape each path component using URL standards
	pathParts := strings.Split(abs, "/")
	escapedParts := slice.Map(pathParts, func(s string) string {
		return url.PathEscape(s)
	})

	u, err := url.Parse(LocalSchemaID + "://" + strings.Join(escapedParts, "/"))
	if err != nil {
		return url.URL{}, err
	}
	return *u, nil
}

// For returns a Storage implementation for the given URI.
// It uses the schema part of the URI to find the correct registered
// Storage constructor.
// If the URI does not contain a schema, it is treated as a file:// URI.
func For(uri string) (Storage, error) {
	lock.RLock()
	defer lock.RUnlock()
	parts := strings.Split(uri, "://")

	var u *url.URL
	// Paths without schema are treated as file:// and use the default LocalStorage implementation
	if len(parts) < 2 {
		parsed, err := LocalPathToURL(uri)
		if err != nil {
			return nil, err
		}
		u = &parsed
	} else {
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		u = parsed
	}

	c, ok := registry[u.Scheme]
	if !ok {
		return nil, errors.New("schema '" + u.Scheme + "' not registered")
	}
	return c(*u), nil
}
