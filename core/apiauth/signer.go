package apiauth

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
)

const Audience = "navidrome-api-v1"

// Tokens minted on one node are verified on others, whose clocks may differ slightly.
const clockSkew = 30 * time.Second

var ErrTokenExpired = errors.New("access token expired")

type claims struct {
	UserID    string
	GrantID   string
	Scopes    []string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type signer struct {
	auth *jwtauth.JWTAuth
}

func newJWTAuth(key []byte, now func() time.Time) *jwtauth.JWTAuth {
	return jwtauth.New("HS256", key, nil,
		jwt.WithAudience(Audience),
		jwt.WithClock(jwt.ClockFunc(now)),
		jwt.WithAcceptableSkew(clockSkew),
		// jwx accepts a token with no exp at all unless the claim is required.
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithRequiredClaim(jwt.SubjectKey),
	)
}

// loadSigner reads the shared v1 key, creating it insert-if-absent so concurrent nodes agree on one key.
func loadSigner(ctx context.Context, ds model.DataStore, now func() time.Time) (*signer, error) {
	key, err := loadKey(ctx, ds)
	if err != nil {
		return nil, err
	}
	return &signer{auth: newJWTAuth([]byte(key), now)}, nil
}

func loadKey(ctx context.Context, ds model.DataStore) (string, error) {
	enc, err := utils.Encrypt(ctx, encryptionKey(), id.NewRandom())
	if err != nil {
		return "", fmt.Errorf("encrypting API v1 key: %w", err)
	}
	if err := ds.Property().PutIfAbsent(ctx, consts.JWTAPIv1SecretKey, enc); err != nil {
		return "", fmt.Errorf("storing API v1 key: %w", err)
	}
	stored, err := ds.Property().Get(ctx, consts.JWTAPIv1SecretKey)
	if err != nil {
		return "", fmt.Errorf("reading API v1 key: %w", err)
	}
	if key, err := utils.Decrypt(ctx, encryptionKey(), stored); err == nil {
		return key, nil
	}
	// A changed PasswordEncryptionKey makes the old key unreadable; replacing it only ends current access tokens.
	// The lock and re-read make concurrent nodes agree on one replacement.
	var key string
	err = ds.WithTxImmediate(func(tx model.DataStore) error {
		current, err := tx.Property().Get(ctx, consts.JWTAPIv1SecretKey)
		if err != nil {
			return err
		}
		if k, err := utils.Decrypt(ctx, encryptionKey(), current); err == nil {
			key = k
			return nil
		}
		log.Warn(ctx, "Could not decrypt API v1 key, replacing it")
		if err := tx.Property().Put(ctx, consts.JWTAPIv1SecretKey, enc); err != nil {
			return err
		}
		key, err = utils.Decrypt(ctx, encryptionKey(), enc)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("replacing API v1 key: %w", err)
	}
	return key, nil
}

func encryptionKey() []byte {
	sum := sha256.Sum256([]byte(cmp.Or(conf.Server.PasswordEncryptionKey, consts.DefaultEncryptionKey)))
	return sum[:]
}

func (s *signer) sign(c claims) (string, error) {
	_, tok, err := s.auth.Encode(map[string]any{
		jwt.SubjectKey:    c.UserID,
		jwt.AudienceKey:   []string{Audience},
		jwt.IssuedAtKey:   c.IssuedAt,
		jwt.ExpirationKey: c.ExpiresAt,
		"gid":             c.GrantID,
		"scope":           strings.Join(c.Scopes, " "),
	})
	return tok, err
}

func (s *signer) parse(token string) (claims, error) {
	tok, err := jwtauth.VerifyToken(s.auth, token)
	if errors.Is(err, jwtauth.ErrExpired) {
		return claims{}, ErrTokenExpired
	}
	if err != nil {
		return claims{}, model.ErrInvalidAuth
	}
	var c claims
	c.UserID, _ = tok.Subject()
	c.IssuedAt, _ = tok.IssuedAt()
	c.ExpiresAt, _ = tok.Expiration()
	var scope string
	if tok.Get("gid", &c.GrantID) != nil || tok.Get("scope", &scope) != nil || c.UserID == "" || c.GrantID == "" {
		return claims{}, model.ErrInvalidAuth
	}
	c.Scopes = strings.Fields(scope)
	return c, nil
}
