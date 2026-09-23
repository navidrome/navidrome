package auth

import (
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Claims represents the typed JWT claims used throughout Navidrome,
// replacing the untyped map[string]any approach.
type Claims struct {
	// Standard JWT claims
	Issuer    string
	Subject   string   // username for session tokens
	Audience  []string // which API may accept this token; empty means any
	IssuedAt  time.Time
	ExpiresAt time.Time

	// Custom claims
	UserID  string // "uid"
	IsAdmin bool   // "adm"
	ID      string // "id" - artwork/mediafile ID
	Format  string // "f" - audio format
	BitRate int    // "b" - audio bitrate
	ShareID string // "sid" - share ID for share stream tokens
	Epoch   int    // "ep" - the user's token_epoch at mint time
}

// ToMap converts Claims to a map[string]any for use with TokenAuth.Encode().
// Only non-zero fields are included.
func (c Claims) ToMap() map[string]any {
	m := make(map[string]any)
	if c.Issuer != "" {
		m[jwt.IssuerKey] = c.Issuer
	}
	if c.Subject != "" {
		m[jwt.SubjectKey] = c.Subject
	}
	if len(c.Audience) > 0 {
		m[jwt.AudienceKey] = c.Audience
	}
	if !c.IssuedAt.IsZero() {
		m[jwt.IssuedAtKey] = c.IssuedAt.UTC().Unix()
	}
	if !c.ExpiresAt.IsZero() {
		m[jwt.ExpirationKey] = c.ExpiresAt.UTC().Unix()
	}
	if c.UserID != "" {
		m["uid"] = c.UserID
	}
	if c.IsAdmin {
		m["adm"] = c.IsAdmin
	}
	if c.ID != "" {
		m["id"] = c.ID
	}
	if c.Format != "" {
		m["f"] = c.Format
	}
	if c.BitRate != 0 {
		m["b"] = c.BitRate
	}
	if c.ShareID != "" {
		m["sid"] = c.ShareID
	}
	if c.Epoch != 0 {
		m["ep"] = c.Epoch
	}
	return m
}

func (c Claims) WithExpiresAt(t time.Time) Claims {
	c.ExpiresAt = t
	return c
}

// ClaimsFromToken extracts Claims directly from a jwt.Token using token.Get().
func ClaimsFromToken(token jwt.Token) Claims {
	var c Claims
	c.Issuer, _ = token.Issuer()
	c.Subject, _ = token.Subject()
	c.IssuedAt, _ = token.IssuedAt()
	c.ExpiresAt, _ = token.Expiration()
	c.Audience, _ = token.Audience()

	var uid string
	if err := token.Get("uid", &uid); err == nil {
		c.UserID = uid
	}
	var adm bool
	if err := token.Get("adm", &adm); err == nil {
		c.IsAdmin = adm
	}
	var id string
	if err := token.Get("id", &id); err == nil {
		c.ID = id
	}
	var f string
	if err := token.Get("f", &f); err == nil {
		c.Format = f
	}
	c.BitRate = intClaim(token, "b")
	var sid string
	if err := token.Get("sid", &sid); err == nil {
		c.ShareID = sid
	}
	c.Epoch = intClaim(token, "ep")
	return c
}

// intClaim reads a numeric claim, which a parsed token may decode as either int or float64.
func intClaim(token jwt.Token, key string) int {
	var i int
	if err := token.Get(key, &i); err == nil {
		return i
	}
	var f float64
	if err := token.Get(key, &f); err == nil {
		return int(f)
	}
	return 0
}
