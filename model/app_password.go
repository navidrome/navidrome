package model

import (
	"context"
	"time"
)

// AppPassword represents a long-lived secondary credential a user creates so
// that Subsonic clients (which cannot perform an OIDC flow) can authenticate
// without exposing the user's primary password. Each app password is bound to
// a single user, has a human-readable name, and is stored AES-GCM-encrypted so
// that the Subsonic md5(password+salt) verification path can read the
// plaintext at request time.
type AppPassword struct {
	ID              string     `structs:"id" json:"id"`
	UserID          string     `structs:"user_id" json:"userId"`
	Name            string     `structs:"name" json:"name"`
	SecretEncrypted string     `structs:"secret_encrypted" json:"-"`
	CreatedAt       time.Time  `structs:"created_at" json:"createdAt"`
	LastUsedAt      *time.Time `structs:"last_used_at" json:"lastUsedAt,omitempty"`
	ExpiresAt       *time.Time `structs:"expires_at" json:"expiresAt,omitempty"`

	// Secret is the plaintext app password. It is populated in two situations
	// only: (a) on the response to Create, where the caller must surface it to
	// the user immediately because it is never recoverable afterwards; and (b)
	// internally inside GetActiveForUser so the Subsonic auth fallback can
	// compute md5(secret+salt). Never persisted, never serialized on read APIs.
	Secret string `structs:"-" json:"secret,omitempty"`
}

type AppPasswords []AppPassword

// AppPasswordPublic is the read-only projection returned by listing endpoints.
// It deliberately omits SecretEncrypted and Secret so neither the ciphertext
// nor the plaintext can leak through generic JSON serialization.
type AppPasswordPublic struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

type AppPasswordRepository interface {
	// Create generates a cryptographically random secret, encrypts it with the
	// shared password-encryption key, persists the record, and returns the
	// plaintext secret along with the new record. The plaintext is returned
	// exactly once; subsequent reads only ever expose the ciphertext.
	//
	// Parameters:
	//   userID    - owner user ID; foreign key into the user table.
	//   name      - human-readable label, unique per user.
	//   expiresAt - optional expiry; nil means the password never expires.
	Create(ctx context.Context, userID, name string, expiresAt *time.Time) (plaintextSecret string, ap *AppPassword, err error)

	// Delete removes the app password identified by id, but only if it is owned
	// by ownerUserID. This double-check prevents users from deleting other
	// users' app passwords by guessing IDs.
	Delete(ctx context.Context, id, ownerUserID string) error

	// GetActiveForUser returns all non-expired app passwords for the given
	// user, with the Secret field populated (decrypted) so the Subsonic
	// authentication fallback can perform md5(secret+salt) checks. Returns an
	// empty slice if the user has no active app passwords.
	GetActiveForUser(ctx context.Context, userID string) (AppPasswords, error)

	// UpdateLastUsedAt sets last_used_at on the record to the current time.
	// Called fire-and-forget after a successful Subsonic authentication, so
	// errors are non-fatal but should be logged.
	UpdateLastUsedAt(ctx context.Context, id string) error

	// ListForUser returns the metadata-only public projection for the given
	// user, suitable for the management UI list view.
	ListForUser(ctx context.Context, userID string) ([]AppPasswordPublic, error)
}
