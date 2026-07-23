package persistence

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
	"github.com/pocketbase/dbx"
)

// appPasswordRepository persists named long-lived app passwords. Each row stores
// an AES-GCM-encrypted secret so that the Subsonic md5(secret+salt) verification
// path can recover the plaintext on each authentication attempt.
type appPasswordRepository struct {
	sqlRepository
}

// appPasswordSecretBytes is the size, in raw bytes, of a generated app-password
// secret before base64 encoding. 24 bytes → 32-character URL-safe string, which
// is well above the practical brute-force threshold for the Subsonic md5+salt
// verification flow.
const appPasswordSecretBytes = 24

// NewAppPasswordRepository constructs a repository over the app_password table.
//
// Side effect: it calls NewUserRepository to guarantee that the shared password
// encryption key (encKey) has been derived and any one-time password
// re-encryption migration has run before this repository is used. Without this
// piggyback, a request that happens to hit /rest before any /api or /auth
// endpoint could observe a nil encKey.
func NewAppPasswordRepository(ctx context.Context, db dbx.Builder) model.AppPasswordRepository {
	_ = NewUserRepository(ctx, db) // ensures encKey is initialised
	r := &appPasswordRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "app_password"
	r.registerModel(&model.AppPassword{}, nil)
	return r
}

// Create generates a fresh secret, encrypts it, and inserts a new row.
//
// Parameters:
//   - ctx        : request context used for cancellation and logging.
//   - userID     : the owning user's ID; must already exist in the user table.
//   - name       : a human-readable label, unique per user.
//   - expiresAt  : optional expiry timestamp; nil means the password never expires.
//
// Returns the plaintext secret (which the caller must surface to the user
// exactly once — it is not retrievable afterwards) plus the persisted record.
func (r *appPasswordRepository) Create(ctx context.Context, userID, name string, expiresAt *time.Time) (string, *model.AppPassword, error) {
	if userID == "" {
		return "", nil, errors.New("appPassword: userID is required")
	}
	if name == "" {
		return "", nil, errors.New("appPassword: name is required")
	}

	plaintext, err := generateAppPasswordSecret()
	if err != nil {
		return "", nil, err
	}
	encrypted, err := utils.Encrypt(ctx, encryptionKey(), plaintext)
	if err != nil {
		return "", nil, err
	}

	ap := &model.AppPassword{
		ID:              id.NewRandom(),
		UserID:          userID,
		Name:            name,
		SecretEncrypted: encrypted,
		CreatedAt:       time.Now(),
		ExpiresAt:       expiresAt,
	}

	insert := Insert(r.tableName).SetMap(map[string]any{
		"id":               ap.ID,
		"user_id":          ap.UserID,
		"name":             ap.Name,
		"secret_encrypted": ap.SecretEncrypted,
		"created_at":       ap.CreatedAt,
		"expires_at":       ap.ExpiresAt,
	})
	if _, err := r.executeSQL(insert); err != nil {
		return "", nil, err
	}
	ap.Secret = plaintext
	return plaintext, ap, nil
}

// Delete removes the row identified by id, but only if owned by ownerUserID.
// The compound WHERE clause prevents users from deleting other users' app
// passwords by guessing IDs.
func (r *appPasswordRepository) Delete(ctx context.Context, id, ownerUserID string) error {
	del := Delete(r.tableName).Where(And{Eq{"id": id}, Eq{"user_id": ownerUserID}})
	count, err := r.executeSQL(del)
	if err != nil {
		return err
	}
	if count == 0 {
		return model.ErrNotFound
	}
	return nil
}

// GetActiveForUser returns every non-expired app password for userID, with the
// Secret field populated by decrypting SecretEncrypted. Used by the Subsonic
// authentication fallback.
func (r *appPasswordRepository) GetActiveForUser(ctx context.Context, userID string) (model.AppPasswords, error) {
	now := time.Now()
	sel := r.newSelect().Columns("id", "user_id", "name", "secret_encrypted", "created_at", "last_used_at", "expires_at").
		Where(Eq{"user_id": userID}).
		Where(Or{Eq{"expires_at": nil}, Gt{"expires_at": now}})
	var rows model.AppPasswords
	if err := r.queryAll(sel, &rows); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.AppPasswords{}, nil
		}
		return nil, err
	}
	for i := range rows {
		plain, err := utils.Decrypt(ctx, encryptionKey(), rows[i].SecretEncrypted)
		if err != nil {
			log.Warn(ctx, "Skipping app password whose secret could not be decrypted", "id", rows[i].ID, err)
			continue
		}
		rows[i].Secret = plain
	}
	return rows, nil
}

// UpdateLastUsedAt sets last_used_at on the row to the current wall clock time.
// Intended to be called fire-and-forget from the Subsonic auth path.
func (r *appPasswordRepository) UpdateLastUsedAt(ctx context.Context, id string) error {
	upd := Update(r.tableName).Where(Eq{"id": id}).Set("last_used_at", time.Now())
	_, err := r.executeSQL(upd)
	return err
}

// ListForUser returns the metadata-only public projection for the user's
// app passwords, sorted by creation time descending so the most recent appears
// first in the management UI.
func (r *appPasswordRepository) ListForUser(ctx context.Context, userID string) ([]model.AppPasswordPublic, error) {
	sel := r.newSelect().
		Columns("id", "user_id", "name", "created_at", "last_used_at", "expires_at").
		Where(Eq{"user_id": userID}).
		OrderBy("created_at DESC")
	var rows []model.AppPasswordPublic
	if err := r.queryAll(sel, &rows); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return []model.AppPasswordPublic{}, nil
		}
		return nil, err
	}
	return rows, nil
}

// generateAppPasswordSecret returns a URL-safe base64-encoded random string
// suitable for use as a secondary password.
func generateAppPasswordSecret() (string, error) {
	buf := make([]byte, appPasswordSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

var _ model.AppPasswordRepository = (*appPasswordRepository)(nil)
