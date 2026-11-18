package model

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// LyricsLocalIdentityVersion is the only local-identity schema this host
// issues or accepts. Local identities address embedded and sidecar lyrics,
// which have no provider recording. They are deliberately separate from
// provider variant identities and from exact original-document selections.
const LyricsLocalIdentityVersion = 1

const (
	LyricsLocalKindEmbedded = "embedded"
	LyricsLocalKindSidecar  = "sidecar"

	MaxLyricsLocalTokenBytes    = 64
	MaxLyricsLocalKindBytes     = 16
	MaxLyricsLocalFormatBytes   = 16
	MaxLyricsLocalLanguageBytes = 35
	MaxLyricsLocalRoleBytes     = 32
)

// LyricsLocalIdentity is an opaque, deterministic address for one selectable
// embedded or sidecar variant. The token hashes the track, source kind,
// playback format and the configured source discriminator; it never contains
// a path, credential or file content. Format and language describe the variant
// for presentation; the token is the address.
type LyricsLocalIdentity struct {
	Version  int    `structs:"version"          json:"version"`
	Token    string `structs:"token"            json:"token"`
	Kind     string `structs:"kind"             json:"kind"`
	Format   string `structs:"format,omitempty" json:"format,omitempty"`
	Language string `structs:"language,omitempty" json:"language,omitempty"`
	Role     string `structs:"role,omitempty"   json:"role,omitempty"`
}

// NewLyricsLocalIdentity derives the host identity for one local source. The
// discriminator is the configured local source pattern (for example ".lrc"),
// which keeps two simultaneously selectable files of the same format apart.
func NewLyricsLocalIdentity(trackID, kind, format, language, role, discriminator string) LyricsLocalIdentity {
	kind = strings.ToLower(strings.TrimSpace(kind))
	format = strings.ToLower(strings.TrimSpace(format))
	language = strings.TrimSpace(language)
	role = strings.TrimSpace(role)
	if role == "" {
		role = LyricKindMain
	}
	identity := LyricsLocalIdentity{
		Version:  LyricsLocalIdentityVersion,
		Kind:     kind,
		Format:   format,
		Language: language,
		Role:     role,
	}
	if strings.TrimSpace(trackID) == "" {
		return identity
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{
		"navidrome-better-lyrics/local-identity/v1",
		trackID,
		kind,
		format,
		strings.TrimSpace(discriminator),
	}, "\x00")))
	identity.Token = fmt.Sprintf("local:v1:%x", digest[:16])
	return identity
}

// Valid reports whether the identity is a bounded host-issued address.
func (identity LyricsLocalIdentity) Valid() bool {
	if identity.Version != LyricsLocalIdentityVersion {
		return false
	}
	if !strings.HasPrefix(identity.Token, "local:v1:") || len(identity.Token) > MaxLyricsLocalTokenBytes {
		return false
	}
	switch identity.Kind {
	case LyricsLocalKindEmbedded, LyricsLocalKindSidecar:
	default:
		return false
	}
	if len(identity.Kind) > MaxLyricsLocalKindBytes ||
		len(identity.Format) > MaxLyricsLocalFormatBytes ||
		len(identity.Language) > MaxLyricsLocalLanguageBytes ||
		len(identity.Role) > MaxLyricsLocalRoleBytes {
		return false
	}
	switch identity.Role {
	case LyricKindMain, LyricKindTranslation, LyricKindPronunciation:
	default:
		return false
	}
	return true
}

// SameAddress compares the opaque address only. Presentation fields may differ
// between a stored preference and a later read of the same local source.
func (identity LyricsLocalIdentity) SameAddress(other LyricsLocalIdentity) bool {
	return identity.Version == other.Version && identity.Token != "" && identity.Token == other.Token
}
