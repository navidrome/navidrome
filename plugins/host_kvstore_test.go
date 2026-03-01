//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KVStoreService", func() {
	var tmpDir string
	var service *kvstoreServiceImpl
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		var err error
		tmpDir, err = os.MkdirTemp("", "kvstore-test-*")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = tmpDir

		// Create service with 1KB limit for testing
		maxSize := "1KB"
		service, err = newKVStoreService("test_plugin", &KVStorePermission{MaxSize: &maxSize})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if service != nil {
			service.Close()
		}
		os.RemoveAll(tmpDir)
	})

	Describe("Basic Operations", func() {
		It("sets and gets a value", func() {
			err := service.Set(ctx, "key1", []byte("value1"))
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.Get(ctx, "key1")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("value1")))
		})

		It("returns not exists for missing key", func() {
			value, exists, err := service.Get(ctx, "missing_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(BeNil())
		})

		It("overwrites existing key", func() {
			err := service.Set(ctx, "key1", []byte("value1"))
			Expect(err).ToNot(HaveOccurred())

			err = service.Set(ctx, "key1", []byte("value2"))
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.Get(ctx, "key1")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("value2")))
		})

		It("handles binary data", func() {
			binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
			err := service.Set(ctx, "binary", binaryData)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.Get(ctx, "binary")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(binaryData))
		})
	})

	Describe("Delete Operation", func() {
		It("deletes a value", func() {
			err := service.Set(ctx, "delete_me", []byte("value"))
			Expect(err).ToNot(HaveOccurred())

			err = service.Delete(ctx, "delete_me")
			Expect(err).ToNot(HaveOccurred())

			_, exists, err := service.Get(ctx, "delete_me")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("does not error when deleting non-existing key", func() {
			err := service.Delete(ctx, "never_existed")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Has Operation", func() {
		It("returns true for existing key", func() {
			err := service.Set(ctx, "exists_key", []byte("value"))
			Expect(err).ToNot(HaveOccurred())

			exists, err := service.Has(ctx, "exists_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false for non-existing key", func() {
			exists, err := service.Has(ctx, "non_existing_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("List Operation", func() {
		BeforeEach(func() {
			Expect(service.Set(ctx, "user:1:name", []byte("Alice"))).To(Succeed())
			Expect(service.Set(ctx, "user:1:email", []byte("alice@test.com"))).To(Succeed())
			Expect(service.Set(ctx, "user:2:name", []byte("Bob"))).To(Succeed())
			Expect(service.Set(ctx, "config:theme", []byte("dark"))).To(Succeed())
		})

		It("lists all keys with empty prefix", func() {
			keys, err := service.List(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(4))
			Expect(keys).To(ContainElements("config:theme", "user:1:email", "user:1:name", "user:2:name"))
		})

		It("lists keys matching prefix", func() {
			keys, err := service.List(ctx, "user:1:")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(2))
			Expect(keys).To(ContainElements("user:1:name", "user:1:email"))
		})

		It("lists keys matching partial prefix", func() {
			keys, err := service.List(ctx, "user:")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(3))
		})

		It("returns empty list for non-matching prefix", func() {
			keys, err := service.List(ctx, "notfound:")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(BeEmpty())
		})

		It("handles special LIKE characters in prefix", func() {
			// Add keys with special characters
			Expect(service.Set(ctx, "test%key", []byte("value1"))).To(Succeed())
			Expect(service.Set(ctx, "test_key", []byte("value2"))).To(Succeed())
			Expect(service.Set(ctx, "testXkey", []byte("value3"))).To(Succeed())

			// Search for "test%"
			keys, err := service.List(ctx, "test%")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(1))
			Expect(keys).To(ContainElement("test%key"))
		})
	})

	Describe("Storage Usage", func() {
		It("reports correct storage used", func() {
			used, err := service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(0)))

			err = service.Set(ctx, "key1", []byte("12345"))
			Expect(err).ToNot(HaveOccurred())

			used, err = service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(5)))

			err = service.Set(ctx, "key2", []byte("67890"))
			Expect(err).ToNot(HaveOccurred())

			used, err = service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(10)))
		})

		It("updates storage when value is overwritten", func() {
			err := service.Set(ctx, "key1", []byte("12345"))
			Expect(err).ToNot(HaveOccurred())

			used, _ := service.GetStorageUsed(ctx)
			Expect(used).To(Equal(int64(5)))

			// Overwrite with smaller value
			err = service.Set(ctx, "key1", []byte("ab"))
			Expect(err).ToNot(HaveOccurred())

			used, _ = service.GetStorageUsed(ctx)
			Expect(used).To(Equal(int64(2)))
		})

		It("decreases storage when key is deleted", func() {
			Expect(service.Set(ctx, "key1", []byte("12345"))).To(Succeed())
			Expect(service.Set(ctx, "key2", []byte("67890"))).To(Succeed())

			used, err := service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(10)))

			Expect(service.Delete(ctx, "key1")).To(Succeed())

			used, err = service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(5)))
		})

		It("updates storage when value is overwritten with larger value", func() {
			err := service.Set(ctx, "key1", []byte("ab"))
			Expect(err).ToNot(HaveOccurred())

			used, _ := service.GetStorageUsed(ctx)
			Expect(used).To(Equal(int64(2)))

			// Overwrite with larger value
			err = service.Set(ctx, "key1", []byte("12345"))
			Expect(err).ToNot(HaveOccurred())

			used, _ = service.GetStorageUsed(ctx)
			Expect(used).To(Equal(int64(5)))
		})

		It("restores correct size after service restart", func() {
			// Add some data
			Expect(service.Set(ctx, "key1", []byte("12345"))).To(Succeed())
			Expect(service.Set(ctx, "key2", []byte("67890"))).To(Succeed())

			used, _ := service.GetStorageUsed(ctx)
			Expect(used).To(Equal(int64(10)))

			// Close and reopen the service (simulating restart)
			Expect(service.Close()).To(Succeed())

			maxSize := "1KB"
			service2, err := newKVStoreService("test_plugin", &KVStorePermission{MaxSize: &maxSize})
			Expect(err).ToNot(HaveOccurred())
			defer service2.Close()

			// Size should be restored from database
			used, err = service2.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(10)))
		})
	})

	Describe("Size Limits", func() {
		It("rejects value when storage limit would be exceeded", func() {
			// Service has 1KB limit
			bigValue := make([]byte, 2048)
			err := service.Set(ctx, "big", bigValue)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage limit exceeded"))
		})

		It("allows updating existing key even if total would exceed limit", func() {
			// Fill up most of the storage
			almostFull := make([]byte, 900)
			err := service.Set(ctx, "big", almostFull)
			Expect(err).ToNot(HaveOccurred())

			// Overwrite with same size should work
			err = service.Set(ctx, "big", almostFull)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Key Validation", func() {
		It("rejects empty key", func() {
			err := service.Set(ctx, "", []byte("value"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key cannot be empty"))
		})

		It("rejects key exceeding max length", func() {
			longKey := strings.Repeat("a", 300)
			err := service.Set(ctx, longKey, []byte("value"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key exceeds maximum length"))
		})
	})

	Describe("Plugin Isolation", func() {
		It("isolates data between plugins", func() {
			service2, err := newKVStoreService("other_plugin", &KVStorePermission{})
			Expect(err).ToNot(HaveOccurred())
			defer service2.Close()

			// Set same key in both plugins
			err = service.Set(ctx, "shared", []byte("value1"))
			Expect(err).ToNot(HaveOccurred())
			err = service2.Set(ctx, "shared", []byte("value2"))
			Expect(err).ToNot(HaveOccurred())

			// Each plugin should get their own value
			val1, _, _ := service.Get(ctx, "shared")
			Expect(val1).To(Equal([]byte("value1")))

			val2, _, _ := service2.Get(ctx, "shared")
			Expect(val2).To(Equal([]byte("value2")))
		})

		It("creates separate database files per plugin", func() {
			service2, err := newKVStoreService("other_plugin", &KVStorePermission{})
			Expect(err).ToNot(HaveOccurred())
			defer service2.Close()

			// Check that separate directories exist
			_, err = os.Stat(filepath.Join(tmpDir, "plugins", "test_plugin", "kvstore.db"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filepath.Join(tmpDir, "plugins", "other_plugin", "kvstore.db"))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Close", func() {
		It("closes database connection", func() {
			err := service.Close()
			Expect(err).ToNot(HaveOccurred())

			// After close, operations should fail
			_, _, err = service.Get(ctx, "any")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Schema Migration", func() {
		It("adds expires_at column to existing databases without it", func() {
			// Close the service that was auto-created in BeforeEach
			service.Close()

			// Create a legacy database manually (without expires_at column)
			dataDir := filepath.Join(tmpDir, "plugins", "legacy_plugin")
			Expect(os.MkdirAll(dataDir, 0700)).To(Succeed())
			dbPath := filepath.Join(dataDir, "kvstore.db")

			legacyDB, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
			Expect(err).ToNot(HaveOccurred())
			_, err = legacyDB.Exec(`
				CREATE TABLE kvstore (
					key TEXT PRIMARY KEY NOT NULL,
					value BLOB NOT NULL,
					size INTEGER NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				)
			`)
			Expect(err).ToNot(HaveOccurred())
			_, err = legacyDB.Exec(`INSERT INTO kvstore (key, value, size) VALUES ('old_key', 'old_value', 9)`)
			Expect(err).ToNot(HaveOccurred())
			legacyDB.Close()

			// Open with new service — should migrate schema
			maxSize := "1KB"
			service, err = newKVStoreService("legacy_plugin", &KVStorePermission{MaxSize: &maxSize})
			Expect(err).ToNot(HaveOccurred())

			// Old data should still be accessible
			value, exists, err := service.Get(ctx, "old_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("old_value")))

			// expires_at should be NULL for old data (no expiration)
			var expiresAt sql.NullTime
			err = service.db.QueryRow(`SELECT expires_at FROM kvstore WHERE key = 'old_key'`).Scan(&expiresAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(expiresAt.Valid).To(BeFalse())
		})
	})

	Describe("TTL Expiration", func() {
		It("Get returns not-exists for expired keys", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('expired_key', 'old', 3, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())
			value, exists, err := service.Get(ctx, "expired_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(BeNil())
		})
		It("Has returns false for expired keys", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('expired_has', 'old', 3, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())
			exists, err := service.Has(ctx, "expired_has")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
		It("List excludes expired keys", func() {
			Expect(service.Set(ctx, "live:1", []byte("alive"))).To(Succeed())
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('live:expired', 'dead', 4, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())
			keys, err := service.List(ctx, "live:")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(1))
			Expect(keys).To(ContainElement("live:1"))
		})
		It("Get returns value for non-expired keys with TTL", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('future_key', 'still alive', 11, datetime('now', '+3600 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())
			value, exists, err := service.Get(ctx, "future_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("still alive")))
		})
		It("Set clears expires_at from a key previously set with TTL", func() {
			err := service.SetWithTTL(ctx, "ttl_then_set", []byte("temp"), 1)
			Expect(err).ToNot(HaveOccurred())

			// Overwrite with Set (no TTL) — should become permanent
			err = service.Set(ctx, "ttl_then_set", []byte("permanent"))
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(2 * time.Second)

			// Should still exist since Set cleared expires_at
			value, exists, err := service.Get(ctx, "ttl_then_set")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("permanent")))
		})
		It("expired keys are not counted in storage used", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('expired_key', '12345', 5, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())

			// Expired keys should not be counted
			used, err := service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(0)))
		})
		It("cleanup removes expired rows from disk", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('cleanup_me', '12345', 5, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())

			// Row exists in DB but is logically expired
			var count int
			Expect(service.db.QueryRow(`SELECT COUNT(*) FROM kvstore`).Scan(&count)).To(Succeed())
			Expect(count).To(Equal(1))

			service.cleanupExpired(ctx)

			// Row should be physically deleted
			Expect(service.db.QueryRow(`SELECT COUNT(*) FROM kvstore`).Scan(&count)).To(Succeed())
			Expect(count).To(Equal(0))
		})
	})

	Describe("SetWithTTL", func() {
		It("stores value that is retrievable before expiry", func() {
			err := service.SetWithTTL(ctx, "ttl_key", []byte("ttl_value"), 3600)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.Get(ctx, "ttl_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("ttl_value")))
		})

		It("value is not retrievable after expiry", func() {
			err := service.SetWithTTL(ctx, "short_ttl", []byte("gone_soon"), 1)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(2 * time.Second)

			_, exists, err := service.Get(ctx, "short_ttl")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("rejects ttlSeconds <= 0", func() {
			err := service.SetWithTTL(ctx, "bad_ttl", []byte("value"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ttlSeconds must be greater than 0"))

			err = service.SetWithTTL(ctx, "bad_ttl", []byte("value"), -5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ttlSeconds must be greater than 0"))
		})

		It("validates key same as Set", func() {
			err := service.SetWithTTL(ctx, "", []byte("value"), 60)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key cannot be empty"))
		})

		It("enforces size limits same as Set", func() {
			bigValue := make([]byte, 2048)
			err := service.SetWithTTL(ctx, "big_ttl", bigValue, 60)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage limit exceeded"))
		})

		It("overwrites existing key and updates TTL", func() {
			err := service.SetWithTTL(ctx, "overwrite_ttl", []byte("first"), 1)
			Expect(err).ToNot(HaveOccurred())

			err = service.SetWithTTL(ctx, "overwrite_ttl", []byte("second"), 3600)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(2 * time.Second)

			value, exists, err := service.Get(ctx, "overwrite_ttl")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal([]byte("second")))
		})

		It("tracks storage correctly", func() {
			err := service.SetWithTTL(ctx, "sized_ttl", []byte("12345"), 3600)
			Expect(err).ToNot(HaveOccurred())

			used, err := service.GetStorageUsed(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(Equal(int64(5)))
		})
	})

	Describe("DeleteByPrefix", func() {
		BeforeEach(func() {
			Expect(service.Set(ctx, "cache:user:1", []byte("Alice"))).To(Succeed())
			Expect(service.Set(ctx, "cache:user:2", []byte("Bob"))).To(Succeed())
			Expect(service.Set(ctx, "cache:item:1", []byte("Widget"))).To(Succeed())
			Expect(service.Set(ctx, "data:important", []byte("keep"))).To(Succeed())
		})

		It("deletes all keys with the given prefix", func() {
			deleted, err := service.DeleteByPrefix(ctx, "cache:user:")
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(int64(2)))

			keys, err := service.List(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(2))
			Expect(keys).To(ContainElements("cache:item:1", "data:important"))
		})

		It("deletes all keys when prefix is empty", func() {
			deleted, err := service.DeleteByPrefix(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(int64(4)))

			keys, err := service.List(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(BeEmpty())
		})

		It("returns 0 when no keys match", func() {
			deleted, err := service.DeleteByPrefix(ctx, "nonexistent:")
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(int64(0)))
		})

		It("updates storage size correctly", func() {
			usedBefore, _ := service.GetStorageUsed(ctx)
			Expect(usedBefore).To(BeNumerically(">", 0))

			_, err := service.DeleteByPrefix(ctx, "cache:")
			Expect(err).ToNot(HaveOccurred())

			usedAfter, _ := service.GetStorageUsed(ctx)
			Expect(usedAfter).To(Equal(int64(4)))
		})

		It("handles special LIKE characters in prefix", func() {
			Expect(service.Set(ctx, "test%special", []byte("v1"))).To(Succeed())
			Expect(service.Set(ctx, "test_special", []byte("v2"))).To(Succeed())
			Expect(service.Set(ctx, "testXspecial", []byte("v3"))).To(Succeed())

			deleted, err := service.DeleteByPrefix(ctx, "test%")
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(int64(1)))

			exists, _ := service.Has(ctx, "test_special")
			Expect(exists).To(BeTrue())
			exists, _ = service.Has(ctx, "testXspecial")
			Expect(exists).To(BeTrue())
		})

		It("also deletes expired keys matching prefix", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('cache:expired', 'old', 3, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())

			deleted, err := service.DeleteByPrefix(ctx, "cache:")
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(int64(4)))
		})
	})

	Describe("GetMany", func() {
		BeforeEach(func() {
			Expect(service.Set(ctx, "key1", []byte("value1"))).To(Succeed())
			Expect(service.Set(ctx, "key2", []byte("value2"))).To(Succeed())
			Expect(service.Set(ctx, "key3", []byte("value3"))).To(Succeed())
		})

		It("retrieves multiple values at once", func() {
			values, err := service.GetMany(ctx, []string{"key1", "key2", "key3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(3))
			Expect(values["key1"]).To(Equal([]byte("value1")))
			Expect(values["key2"]).To(Equal([]byte("value2")))
			Expect(values["key3"]).To(Equal([]byte("value3")))
		})

		It("omits missing keys from result", func() {
			values, err := service.GetMany(ctx, []string{"key1", "missing", "key3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(2))
			Expect(values["key1"]).To(Equal([]byte("value1")))
			Expect(values["key3"]).To(Equal([]byte("value3")))
			_, hasMissing := values["missing"]
			Expect(hasMissing).To(BeFalse())
		})

		It("returns empty map for empty keys slice", func() {
			values, err := service.GetMany(ctx, []string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(BeEmpty())
		})

		It("returns empty map for nil keys slice", func() {
			values, err := service.GetMany(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(BeEmpty())
		})

		It("excludes expired keys", func() {
			_, err := service.db.Exec(`
				INSERT INTO kvstore (key, value, size, expires_at)
				VALUES ('expired_many', 'old', 3, datetime('now', '-1 seconds'))
			`)
			Expect(err).ToNot(HaveOccurred())

			values, err := service.GetMany(ctx, []string{"key1", "expired_many"})
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(1))
			Expect(values["key1"]).To(Equal([]byte("value1")))
		})

		It("handles all keys missing", func() {
			values, err := service.GetMany(ctx, []string{"nope1", "nope2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(BeEmpty())
		})
	})
})

var _ = Describe("KVStoreService Integration", Ordered, func() {
	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "kvstore-integration-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-kvstore plugin
		srcPath := filepath.Join(testdataDir, "test-kvstore"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-kvstore"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")
		conf.Server.DataFolder = tmpDir

		// Setup mock DataStore with pre-enabled plugin
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-kvstore",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
		}})
		dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("should load plugin with kvstore permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-kvstore"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Kvstore).ToNot(BeNil())
			Expect(*p.manifest.Permissions.Kvstore.MaxSize).To(Equal("10KB"))
		})
	})

	Describe("KVStore Operations via Plugin", func() {
		type testKVStoreInput struct {
			Operation  string   `json:"operation"`
			Key        string   `json:"key"`
			Value      []byte   `json:"value,omitempty"`
			Prefix     string   `json:"prefix,omitempty"`
			TTLSeconds int64    `json:"ttl_seconds,omitempty"`
			Keys       []string `json:"keys,omitempty"`
		}
		type testKVStoreOutput struct {
			Value        []byte            `json:"value,omitempty"`
			Values       map[string][]byte `json:"values,omitempty"`
			Exists       bool              `json:"exists,omitempty"`
			Keys         []string          `json:"keys,omitempty"`
			StorageUsed  int64             `json:"storage_used,omitempty"`
			DeletedCount int64             `json:"deleted_count,omitempty"`
			Error        *string           `json:"error,omitempty"`
		}

		callTestKVStore := func(ctx context.Context, input testKVStoreInput) (*testKVStoreOutput, error) {
			manager.mu.RLock()
			p := manager.plugins["test-kvstore"]
			manager.mu.RUnlock()

			instance, err := p.instance(ctx)
			if err != nil {
				return nil, err
			}
			defer instance.Close(ctx)

			inputBytes, _ := json.Marshal(input)
			_, outputBytes, err := instance.Call("nd_test_kvstore", inputBytes)
			if err != nil {
				return nil, err
			}

			var output testKVStoreOutput
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				return nil, err
			}
			if output.Error != nil {
				return nil, errors.New(*output.Error)
			}
			return &output, nil
		}

		It("should set and get value", func() {
			ctx := GinkgoT().Context()

			// Set value
			_, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "set",
				Key:       "test_key",
				Value:     []byte("hello kvstore"),
			})
			Expect(err).ToNot(HaveOccurred())

			// Get value
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "get",
				Key:       "test_key",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.Value).To(Equal([]byte("hello kvstore")))
		})

		It("should check key existence with has", func() {
			ctx := GinkgoT().Context()

			// Check existing key
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "has",
				Key:       "test_key",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())

			// Check non-existing key
			output, err = callTestKVStore(ctx, testKVStoreInput{
				Operation: "has",
				Key:       "non_existing",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should delete value", func() {
			ctx := GinkgoT().Context()

			// Set another key
			_, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "set",
				Key:       "to_delete",
				Value:     []byte("delete me"),
			})
			Expect(err).ToNot(HaveOccurred())

			// Delete it
			_, err = callTestKVStore(ctx, testKVStoreInput{
				Operation: "delete",
				Key:       "to_delete",
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify it's gone
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "has",
				Key:       "to_delete",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should list keys with prefix", func() {
			ctx := GinkgoT().Context()

			// Set some keys
			for _, key := range []string{"prefix:1", "prefix:2", "other:1"} {
				_, err := callTestKVStore(ctx, testKVStoreInput{
					Operation: "set",
					Key:       key,
					Value:     []byte("value"),
				})
				Expect(err).ToNot(HaveOccurred())
			}

			// List with prefix
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "list",
				Prefix:    "prefix:",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Keys).To(HaveLen(2))
			Expect(output.Keys).To(ContainElements("prefix:1", "prefix:2"))
		})

		It("should report storage used", func() {
			ctx := GinkgoT().Context()

			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "get_storage_used",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.StorageUsed).To(BeNumerically(">", 0))
		})

		It("should enforce size limits", func() {
			ctx := GinkgoT().Context()

			// Plugin has 10KB limit, try to exceed it
			bigValue := make([]byte, 15*1024)
			_, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "set",
				Key:       "too_big",
				Value:     bigValue,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage limit exceeded"))
		})

		It("should handle binary data with null bytes through WASM", func() {
			ctx := GinkgoT().Context()

			// Binary data with null bytes, high bytes, and other edge cases
			binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x80, 0x7F}

			// Set binary value
			_, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "set",
				Key:       "binary_test",
				Value:     binaryData,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get binary value and verify exact match
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "get",
				Key:       "binary_test",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.Value).To(Equal(binaryData))
		})

		It("should set value with TTL and expire it", func() {
			ctx := GinkgoT().Context()

			// Set value with 1 second TTL
			_, err := callTestKVStore(ctx, testKVStoreInput{
				Operation:  "set_with_ttl",
				Key:        "ttl_key",
				Value:      []byte("temporary"),
				TTLSeconds: 1,
			})
			Expect(err).ToNot(HaveOccurred())

			// Immediately should exist
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "get",
				Key:       "ttl_key",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.Value).To(Equal([]byte("temporary")))

			// Wait for expiration
			time.Sleep(2 * time.Second)

			// Should no longer exist
			output, err = callTestKVStore(ctx, testKVStoreInput{
				Operation: "get",
				Key:       "ttl_key",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should delete keys by prefix", func() {
			ctx := GinkgoT().Context()

			// Set multiple keys with shared prefix
			for _, key := range []string{"del_prefix:a", "del_prefix:b", "keep:c"} {
				_, err := callTestKVStore(ctx, testKVStoreInput{
					Operation: "set",
					Key:       key,
					Value:     []byte("value"),
				})
				Expect(err).ToNot(HaveOccurred())
			}

			// Delete by prefix
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "delete_by_prefix",
				Prefix:    "del_prefix:",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.DeletedCount).To(Equal(int64(2)))

			// Verify remaining key
			getOutput, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "has",
				Key:       "keep:c",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(getOutput.Exists).To(BeTrue())

			// Verify deleted keys are gone
			getOutput, err = callTestKVStore(ctx, testKVStoreInput{
				Operation: "has",
				Key:       "del_prefix:a",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(getOutput.Exists).To(BeFalse())
		})

		It("should get many values at once", func() {
			ctx := GinkgoT().Context()

			// Set multiple keys
			for _, kv := range []struct{ k, v string }{
				{"many:1", "val1"},
				{"many:2", "val2"},
				{"many:3", "val3"},
			} {
				_, err := callTestKVStore(ctx, testKVStoreInput{
					Operation: "set",
					Key:       kv.k,
					Value:     []byte(kv.v),
				})
				Expect(err).ToNot(HaveOccurred())
			}

			// Get many, including a missing key
			output, err := callTestKVStore(ctx, testKVStoreInput{
				Operation: "get_many",
				Keys:      []string{"many:1", "many:3", "many:missing"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Values).To(HaveLen(2))
			Expect(output.Values["many:1"]).To(Equal([]byte("val1")))
			Expect(output.Values["many:3"]).To(Equal([]byte("val3")))
			_, hasMissing := output.Values["many:missing"]
			Expect(hasMissing).To(BeFalse())
		})
	})

	Describe("Database Isolation", func() {
		It("should create separate database file for plugin", func() {
			dbPath := filepath.Join(tmpDir, "plugins", "test-kvstore", "kvstore.db")
			_, err := os.Stat(dbPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
