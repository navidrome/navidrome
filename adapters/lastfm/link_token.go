package lastfm

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/core/auth"
)

const (
	linkTokenScope = "lastfm-link"
	linkTokenTTL   = 5 * time.Minute
)

// createLinkToken issues a short-lived, signed token that binds the Last.fm
// link-callback to the authenticated user who initiated the OAuth flow. The
// resulting opaque value is passed back through Last.fm via the `cb` URL and
// verified on the callback, replacing the previously-trusted raw `uid` query
// parameter.
func createLinkToken(userID string) (string, error) {
	claims := map[string]any{
		"uid":   userID,
		"scope": linkTokenScope,
		"exp":   time.Now().Add(linkTokenTTL).UTC().Unix(),
	}
	return auth.EncodeToken(claims)
}

// verifyLinkToken validates a signed link token and returns the encoded user ID.
// It enforces both the signature/expiry (via the underlying JWT verifier) and a
// dedicated scope claim, preventing tokens minted for other purposes (e.g. a
// regular session JWT) from being accepted here.
func verifyLinkToken(tokenStr string) (string, error) {
	token, err := auth.DecodeAndVerifyToken(tokenStr)
	if err != nil {
		return "", err
	}
	var scope string
	if err := token.Get("scope", &scope); err != nil || scope != linkTokenScope {
		return "", errors.New("invalid link token scope")
	}
	var uid string
	if err := token.Get("uid", &uid); err != nil || uid == "" {
		return "", errors.New("invalid link token subject")
	}
	return uid, nil
}
