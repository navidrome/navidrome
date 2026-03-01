package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/slice"
)

const (
	defaultMaxKVStoreSize = 1 * 1024 * 1024 // 1MB default
	maxKeyLength          = 256             // Max key length in bytes
)

// notExpiredFilter is the SQL condition to exclude expired keys.
const notExpiredFilter = "(expires_at IS NULL OR expires_at > datetime('now'))"

const cleanupInterval = 1 * time.Hour

// kvstoreServiceImpl implements the host.KVStoreService interface.
// Each plugin gets its own SQLite database for isolation.
type kvstoreServiceImpl struct {
	pluginName string
	db         *sql.DB
	maxSize    int64
}

// newKVStoreService creates a new kvstoreServiceImpl instance with its own SQLite database.
// The provided context controls the lifetime of the background cleanup goroutine.
func newKVStoreService(ctx context.Context, pluginName string, perm *KVStorePermission) (*kvstoreServiceImpl, error) {
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

	// Apply schema migrations
	if err := createKVStoreSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating kvstore schema: %w", err)
	}

	log.Debug("Initialized plugin kvstore", "plugin", pluginName, "path", dbPath, "maxSize", humanize.Bytes(uint64(maxSize)))

	svc := &kvstoreServiceImpl{
		pluginName: pluginName,
		db:         db,
		maxSize:    maxSize,
	}
	go svc.cleanupLoop(ctx)
	return svc, nil
}

// createKVStoreSchema applies schema migrations to the kvstore database.
// New migrations must be appended at the end of the slice.
func createKVStoreSchema(db *sql.DB) error {
	return migrateDB(db, []string{
		`CREATE TABLE IF NOT EXISTS kvstore (
			key TEXT PRIMARY KEY NOT NULL,
			value BLOB NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE kvstore ADD COLUMN expires_at DATETIME DEFAULT NULL`,
		`CREATE INDEX idx_kvstore_expires_at ON kvstore(expires_at)`,
	})
}

// storageUsed returns the current total storage used by non-expired keys.
func (s *kvstoreServiceImpl) storageUsed(ctx context.Context) (int64, error) {
	var used int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size), 0) FROM kvstore WHERE `+notExpiredFilter).Scan(&used)
	if err != nil {
		return 0, fmt.Errorf("calculating storage used: %w", err)
	}
	return used, nil
}

// checkStorageLimit verifies that adding delta bytes would not exceed the storage limit.
func (s *kvstoreServiceImpl) checkStorageLimit(ctx context.Context, delta int64) error {
	if delta <= 0 {
		return nil
	}
	used, err := s.storageUsed(ctx)
	if err != nil {
		return err
	}
	newTotal := used + delta
	if newTotal > s.maxSize {
		return fmt.Errorf("storage limit exceeded: would use %s of %s allowed",
			humanize.Bytes(uint64(newTotal)), humanize.Bytes(uint64(s.maxSize)))
	}
	return nil
}

// setValue is the shared implementation for Set and SetWithTTL.
// A ttlSeconds of 0 means no expiration.
func (s *kvstoreServiceImpl) setValue(ctx context.Context, key string, value []byte, ttlSeconds int64) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > maxKeyLength {
		return fmt.Errorf("key exceeds maximum length of %d bytes", maxKeyLength)
	}

	newValueSize := int64(len(value))

	// Get current size of this key (if it exists and not expired) to calculate delta
	var oldSize int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(size, 0) FROM kvstore WHERE key = ? AND `+notExpiredFilter, key).Scan(&oldSize)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking existing key: %w", err)
	}

	if err := s.checkStorageLimit(ctx, newValueSize-oldSize); err != nil {
		return err
	}

	// Compute expires_at: sql.NullString{Valid:false} sends NULL (no expiration),
	// otherwise we send a concrete timestamp.
	var expiresAt sql.NullString
	if ttlSeconds > 0 {
		expiresAt = sql.NullString{String: fmt.Sprintf("+%d seconds", ttlSeconds), Valid: true}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO kvstore (key, value, size, created_at, updated_at, expires_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, datetime('now', ?))
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			size = excluded.size,
			updated_at = CURRENT_TIMESTAMP,
			expires_at = excluded.expires_at
	`, key, value, newValueSize, expiresAt)
	if err != nil {
		return fmt.Errorf("storing value: %w", err)
	}

	log.Trace(ctx, "KVStore.Set", "plugin", s.pluginName, "key", key, "size", newValueSize, "ttlSeconds", ttlSeconds)
	return nil
}

// Set stores a byte value with the given key.
func (s *kvstoreServiceImpl) Set(ctx context.Context, key string, value []byte) error {
	return s.setValue(ctx, key, value, 0)
}

// SetWithTTL stores a byte value with the given key and a time-to-live.
func (s *kvstoreServiceImpl) SetWithTTL(ctx context.Context, key string, value []byte, ttlSeconds int64) error {
	if ttlSeconds <= 0 {
		return fmt.Errorf("ttlSeconds must be greater than 0")
	}
	return s.setValue(ctx, key, value, ttlSeconds)
}

// Get retrieves a byte value from storage.
func (s *kvstoreServiceImpl) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	err := s.db.QueryRowContext(ctx, `SELECT value FROM kvstore WHERE key = ? AND `+notExpiredFilter, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
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
	_, err := s.db.ExecContext(ctx, `DELETE FROM kvstore WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("deleting value: %w", err)
	}

	log.Trace(ctx, "KVStore.Delete", "plugin", s.pluginName, "key", key)
	return nil
}

// Has checks if a key exists in storage.
func (s *kvstoreServiceImpl) Has(ctx context.Context, key string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM kvstore WHERE key = ? AND `+notExpiredFilter, key).Scan(&count)
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
		rows, err = s.db.QueryContext(ctx, `SELECT key FROM kvstore WHERE `+notExpiredFilter+` ORDER BY key`)
	} else {
		// Escape special LIKE characters in prefix
		escapedPrefix := strings.ReplaceAll(prefix, "%", "\\%")
		escapedPrefix = strings.ReplaceAll(escapedPrefix, "_", "\\_")
		rows, err = s.db.QueryContext(ctx, `SELECT key FROM kvstore WHERE key LIKE ? ESCAPE '\' AND `+notExpiredFilter+` ORDER BY key`, escapedPrefix+"%")
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
	used, err := s.storageUsed(ctx)
	if err != nil {
		return 0, err
	}
	log.Trace(ctx, "KVStore.GetStorageUsed", "plugin", s.pluginName, "bytes", used)
	return used, nil
}

// DeleteByPrefix removes all keys matching the given prefix.
func (s *kvstoreServiceImpl) DeleteByPrefix(ctx context.Context, prefix string) (int64, error) {
	if prefix == "" {
		return 0, fmt.Errorf("prefix cannot be empty")
	}

	escapedPrefix := strings.ReplaceAll(prefix, "%", "\\%")
	escapedPrefix = strings.ReplaceAll(escapedPrefix, "_", "\\_")
	result, err := s.db.ExecContext(ctx, `DELETE FROM kvstore WHERE key LIKE ? ESCAPE '\'`, escapedPrefix+"%")
	if err != nil {
		return 0, fmt.Errorf("deleting keys: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting deleted count: %w", err)
	}

	log.Trace(ctx, "KVStore.DeleteByPrefix", "plugin", s.pluginName, "prefix", prefix, "deletedCount", count)
	return count, nil
}

// GetMany retrieves multiple values in a single call, processing keys in batches.
func (s *kvstoreServiceImpl) GetMany(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return map[string][]byte{}, nil
	}

	const batchSize = 200
	result := make(map[string][]byte)
	for chunk := range slice.CollectChunks(slices.Values(keys), batchSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, key := range chunk {
			placeholders[i] = "?"
			args[i] = key
		}

		query := `SELECT key, value FROM kvstore WHERE key IN (` + strings.Join(placeholders, ",") + `) AND ` + notExpiredFilter //nolint:gosec // placeholders are always "?"
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("querying values: %w", err)
		}

		for rows.Next() {
			var key string
			var value []byte
			if err := rows.Scan(&key, &value); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scanning value: %w", err)
			}
			result[key] = value
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterating values: %w", err)
		}
		rows.Close()
	}

	log.Trace(ctx, "KVStore.GetMany", "plugin", s.pluginName, "requested", len(keys), "found", len(result))
	return result, nil
}

// cleanupLoop periodically removes expired keys from the database.
// It stops when the provided context is cancelled.
func (s *kvstoreServiceImpl) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpired(ctx)
		}
	}
}

// cleanupExpired removes all expired keys from the database to reclaim disk space.
func (s *kvstoreServiceImpl) cleanupExpired(ctx context.Context) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM kvstore WHERE expires_at IS NOT NULL AND expires_at <= datetime('now')`)
	if err != nil {
		log.Error(ctx, "KVStore cleanup: failed to delete expired keys", "plugin", s.pluginName, err)
		return
	}
	if count, err := result.RowsAffected(); err == nil && count > 0 {
		log.Debug("KVStore cleanup completed", "plugin", s.pluginName, "deletedKeys", count)
	}
}

// Close runs a final cleanup and closes the SQLite database connection.
// The cleanup goroutine is stopped by the context passed to newKVStoreService.
func (s *kvstoreServiceImpl) Close() error {
	if s.db != nil {
		log.Debug("Closing plugin kvstore", "plugin", s.pluginName)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cleanupExpired(ctx)
		return s.db.Close()
	}
	return nil
}

// Compile-time verification
var _ host.KVStoreService = (*kvstoreServiceImpl)(nil)
