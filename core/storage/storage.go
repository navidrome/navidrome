package storage

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
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

// For returns a Storage implementation for the given URI.
// It uses the schema part of the URI to find the correct registered
// Storage constructor.
// If the URI does not contain a schema, it is treated as a file:// URI.
func For(uri string) (Storage, error) {
	lock.RLock()
	defer lock.RUnlock()
	parts := strings.Split(uri, "://")

	// Paths without schema are treated as file:// and use the default LocalStorage implementation
	if len(parts) < 2 {
		uri, _ = filepath.Abs(uri)
		uri = LocalSchemaID + "://" + uri
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	c, ok := registry[u.Scheme]
	if !ok {
		return nil, errors.New("schema '" + u.Scheme + "' not registered")
	}
	return c(*u), nil
}
