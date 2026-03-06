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
	Subject   string // username for session tokens
	IssuedAt  time.Time
	ExpiresAt time.Time

	// Custom claims
	UserID  string // "uid"
	IsAdmin bool   // "adm"
	ID      string // "id" - artwork/mediafile ID
	Format  string // "f" - audio format
	BitRate int    // "b" - audio bitrate

	// Transcoding claims
	MediaID    string // "mid" - media file ID
	DirectPlay bool   // "dp" - direct play flag
	UpdatedAt  int64  // "ua" - source file updated-at (Unix seconds)
	Channels   int    // "ch" - target channels
	SampleRate int    // "sr" - target sample rate (Hz)
	BitDepth   int    // "bd" - target bit depth
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
	if c.MediaID != "" {
		m["mid"] = c.MediaID
	}
	if c.DirectPlay {
		m["dp"] = c.DirectPlay
	}
	if c.UpdatedAt != 0 {
		m["ua"] = c.UpdatedAt
	}
	if c.Channels != 0 {
		m["ch"] = c.Channels
	}
	if c.SampleRate != 0 {
		m["sr"] = c.SampleRate
	}
	if c.BitDepth != 0 {
		m["bd"] = c.BitDepth
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
	if err := token.Get("b", &c.BitRate); err != nil {
		var bf float64
		if err := token.Get("b", &bf); err == nil {
			c.BitRate = int(bf)
		}
	}
	var mid string
	if err := token.Get("mid", &mid); err == nil {
		c.MediaID = mid
	}
	var dp bool
	if err := token.Get("dp", &dp); err == nil {
		c.DirectPlay = dp
	}
	if err := token.Get("ua", &c.UpdatedAt); err != nil {
		var uaf float64
		if err := token.Get("ua", &uaf); err == nil {
			c.UpdatedAt = int64(uaf)
		}
	}
	if err := token.Get("ch", &c.Channels); err != nil {
		var chf float64
		if err := token.Get("ch", &chf); err == nil {
			c.Channels = int(chf)
		}
	}
	if err := token.Get("sr", &c.SampleRate); err != nil {
		var srf float64
		if err := token.Get("sr", &srf); err == nil {
			c.SampleRate = int(srf)
		}
	}
	if err := token.Get("bd", &c.BitDepth); err != nil {
		var bdf float64
		if err := token.Get("bd", &bdf); err == nil {
			c.BitDepth = int(bdf)
		}
	}
	return c
}
