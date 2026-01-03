package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/dustin/go-humanize"
	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
)

const (
	defaultMaxKVStoreSize = 1 * 1024 * 1024 // 1MB default
	maxKeyLength          = 256             // Max key length in bytes
)

// kvstoreServiceImpl implements the host.KVStoreService interface.
// Each plugin gets its own SQLite database for isolation.
type kvstoreServiceImpl struct {
	pluginName  string
	db          *sql.DB
	maxSize     int64
	currentSize atomic.Int64 // cached total size, updated on Set/Delete
}

// newKVStoreService creates a new kvstoreServiceImpl instance with its own SQLite database.
func newKVStoreService(pluginName string, perm *KVStorePermission) (*kvstoreServiceImpl, error) {
	// Parse max size from permission, default to 1MB
	maxSize := int64(defaultMaxKVStoreSize)
	if perm != nil && perm.MaxSize != nil && *perm.MaxSize != "" {
		parsed, err := humanize.ParseBytes(*perm.MaxSize)
		if err != nil {
			return nil, fmt.Errorf("invalid maxSize %q: %w", *perm.MaxSize, err)
		}
		maxSize = int64(parsed)
	}

	// Create plugin data directory
	dataDir := filepath.Join(conf.Server.DataFolder, "plugins", pluginName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating plugin data directory: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(dataDir, "kvstore.db")
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=off")
	if err != nil {
		return nil, fmt.Errorf("opening kvstore database: %w", err)
	}

	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)

	// Create schema
	if err := createKVStoreSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating kvstore schema: %w", err)
	}

	// Load current storage size from database
	var currentSize int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(size), 0) FROM kvstore`).Scan(&currentSize); err != nil {
		db.Close()
		return nil, fmt.Errorf("loading storage size: %w", err)
	}

	log.Debug("Initialized plugin kvstore", "plugin", pluginName, "path", dbPath, "maxSize", humanize.Bytes(uint64(maxSize)), "currentSize", humanize.Bytes(uint64(currentSize)))

	svc := &kvstoreServiceImpl{
		pluginName: pluginName,
		db:         db,
		maxSize:    maxSize,
	}
	svc.currentSize.Store(currentSize)
	return svc, nil
}

func createKVStoreSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kvstore (
			key TEXT PRIMARY KEY NOT NULL,
			value BLOB NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// Set stores a byte value with the given key.
func (s *kvstoreServiceImpl) Set(ctx context.Context, key string, value []byte) error {
	// Validate key
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > maxKeyLength {
		return fmt.Errorf("key exceeds maximum length of %d bytes", maxKeyLength)
	}

	newValueSize := int64(len(value))

	// Get current size of this key (if it exists) to calculate delta
	var oldSize int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(size, 0) FROM kvstore WHERE key = ?`, key).Scan(&oldSize)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking existing key: %w", err)
	}

	// Check size limits using cached total
	delta := newValueSize - oldSize
	newTotal := s.currentSize.Load() + delta
	if newTotal > s.maxSize {
		return fmt.Errorf("storage limit exceeded: would use %s of %s allowed",
			humanize.Bytes(uint64(newTotal)), humanize.Bytes(uint64(s.maxSize)))
	}

	// Upsert the value
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO kvstore (key, value, size, created_at, updated_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET 
			value = excluded.value, 
			size = excluded.size,
			updated_at = CURRENT_TIMESTAMP
	`, key, value, newValueSize)
	if err != nil {
		return fmt.Errorf("storing value: %w", err)
	}

	// Update cached size
	s.currentSize.Add(delta)

	log.Trace(ctx, "KVStore.Set", "plugin", s.pluginName, "key", key, "size", newValueSize)
	return nil
}

// Get retrieves a byte value from storage.
func (s *kvstoreServiceImpl) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	err := s.db.QueryRowContext(ctx, `SELECT value FROM kvstore WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading value: %w", err)
	}

	log.Trace(ctx, "KVStore.Get", "plugin", s.pluginName, "key", key, "found", true)
	return value, true, nil
}

// Delete removes a value from storage.
func (s *kvstoreServiceImpl) Delete(ctx context.Context, key string) error {
	// Get size of the key being deleted to update cache
	var oldSize int64
	err := s.db.QueryRowContext(ctx, `SELECT size FROM kvstore WHERE key = ?`, key).Scan(&oldSize)
	if errors.Is(err, sql.ErrNoRows) {
		// Key doesn't exist, nothing to delete
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking key size: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM kvstore WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("deleting value: %w", err)
	}

	// Update cached size
	s.currentSize.Add(-oldSize)

	log.Trace(ctx, "KVStore.Delete", "plugin", s.pluginName, "key", key)
	return nil
}

// Has checks if a key exists in storage.
func (s *kvstoreServiceImpl) Has(ctx context.Context, key string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM kvstore WHERE key = ?`, key).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking key: %w", err)
	}

	return count > 0, nil
}

// List returns all keys matching the given prefix.
func (s *kvstoreServiceImpl) List(ctx context.Context, prefix string) ([]string, error) {
	var rows *sql.Rows
	var err error

	if prefix == "" {
		rows, err = s.db.QueryContext(ctx, `SELECT key FROM kvstore ORDER BY key`)
	} else {
		// Escape special LIKE characters in prefix
		escapedPrefix := strings.ReplaceAll(prefix, "%", "\\%")
		escapedPrefix = strings.ReplaceAll(escapedPrefix, "_", "\\_")
		rows, err = s.db.QueryContext(ctx, `SELECT key FROM kvstore WHERE key LIKE ? ESCAPE '\' ORDER BY key`, escapedPrefix+"%")
	}
	if err != nil {
		return nil, fmt.Errorf("listing keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating keys: %w", err)
	}

	log.Trace(ctx, "KVStore.List", "plugin", s.pluginName, "prefix", prefix, "count", len(keys))
	return keys, nil
}

// GetStorageUsed returns the total storage used by this plugin in bytes.
func (s *kvstoreServiceImpl) GetStorageUsed(ctx context.Context) (int64, error) {
	used := s.currentSize.Load()
	log.Trace(ctx, "KVStore.GetStorageUsed", "plugin", s.pluginName, "bytes", used)
	return used, nil
}

// Close closes the SQLite database connection.
// This is called when the plugin is unloaded.
func (s *kvstoreServiceImpl) Close() error {
	if s.db != nil {
		log.Debug("Closing plugin kvstore", "plugin", s.pluginName)
		return s.db.Close()
	}
	return nil
}

// Compile-time verification
var _ host.KVStoreService = (*kvstoreServiceImpl)(nil)
