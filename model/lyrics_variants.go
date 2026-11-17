package model

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// LyricsVariantVersion is the only identity schema Navidrome accepts.
const LyricsVariantVersion = 1

const (
	MaxLyricsVariantInstanceBytes  = 192
	MaxLyricsVariantRecordingBytes = 128
	MaxLyricsVariantLanguageBytes  = 35
	MaxLyricsVariantRoleBytes      = 32
	MaxLyricsVariantFormatBytes    = 16
	MaxLyricsVariantPluginBytes    = 128
	// MaxLyricsVariantsPerPlugin bounds one plugin's listing, matching the
	// plugin's shared playback/listing bound.
	MaxLyricsVariantsPerPlugin = 64
	// MaxLyricsVariants bounds the global listing across configured plugins
	// (bounded plugin patterns × per-plugin bound). Dropped selections stay
	// absent (and therefore unavailable); they are never remapped.
	MaxLyricsVariants = 512
	// MaxLyricsOriginalBytes matches the plugin's per-entry cache bound.
	MaxLyricsOriginalBytes  = 1 << 20
	MaxLyricsSelectionBytes = 1024
)

var ErrLyricsOriginalUnavailable = errors.New("lyrics original unavailable")

// LyricsVariantIdentity names one addressable playback representation.
type LyricsVariantIdentity struct {
	Version   int    `structs:"version"             json:"version"`
	Instance  string `structs:"instance,omitempty"  json:"instance,omitempty"`
	Recording string `structs:"recording,omitempty" json:"recording,omitempty"`
	Language  string `structs:"language,omitempty"  json:"language,omitempty"`
	Role      string `structs:"role,omitempty"      json:"role,omitempty"`
	Format    string `structs:"format,omitempty"    json:"format,omitempty"`
}

// Valid reports whether an identity is a supported, bounded, exact address.
func (identity LyricsVariantIdentity) Valid() bool {
	if identity.Version != LyricsVariantVersion {
		return false
	}
	if identity.Instance == "" || identity.Recording == "" || identity.Format == "" {
		return false
	}
	if len(identity.Instance) > MaxLyricsVariantInstanceBytes ||
		len(identity.Recording) > MaxLyricsVariantRecordingBytes ||
		len(identity.Language) > MaxLyricsVariantLanguageBytes ||
		len(identity.Role) > MaxLyricsVariantRoleBytes ||
		len(identity.Format) > MaxLyricsVariantFormatBytes {
		return false
	}
	switch identity.Role {
	case LyricKindMain, LyricKindTranslation, LyricKindPronunciation:
	default:
		return false
	}
	return lyricsPlaybackFormat(identity.Format)
}

// lyricsPlaybackFormat is the set of representations Navidrome can parse for
// playback. An identity names a playback entry; a native-only document format
// such as QRC is addressed by the entry it was converted into.
func lyricsPlaybackFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "ttml", "lrc", "elrc", "plain":
		return true
	}
	return false
}

// Key is a stable dedupe key for one identity.
func (identity LyricsVariantIdentity) Key() string {
	return strings.Join([]string{
		identity.Instance, identity.Recording, identity.Language, identity.Role, identity.Format,
	}, "\x00")
}

// LyricsOriginalInfo reports whether an exact native document is retrievable.
type LyricsOriginalInfo struct {
	Available  bool   `json:"available"`
	Format     string `json:"format,omitempty"`
	Provenance string `json:"provenance,omitempty"`
	MediaType  string `json:"mediaType,omitempty"`
	Extension  string `json:"extension,omitempty"`
	Size       int    `json:"size,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

const (
	LyricsMatchUnassessed = "unassessed"
	LyricsMatchValidated  = "validated"
	LyricsMatchAmbiguous  = "ambiguous"
	LyricsMatchMismatch   = "mismatch"
)

// LyricsMatch is the recording-identity assessment, independent of timing
// richness and content state. AutoSaveEligible is true only for validated.
type LyricsMatch struct {
	Status        string   `structs:"status"                     json:"status"`
	Reasons       []string `structs:"reasons,omitempty"          json:"reasons,omitempty"`
	DurationDelta float64  `structs:"durationDelta,omitempty"    json:"durationDelta,omitempty"`
	Recording     string   `structs:"recording,omitempty"        json:"recording,omitempty"`
	AutoSave      bool     `structs:"autoSaveEligible"           json:"autoSaveEligible"`
}

// LyricsEvidence is a typed, attributed provider observation. It is separate
// from playable lyric entries and is never inferred from empty/error responses.
type LyricsEvidence struct {
	PluginID  string       `structs:"pluginId,omitempty"  json:"pluginId,omitempty"`
	Kind      string       `structs:"kind"                json:"kind"`
	Provider  string       `structs:"provider,omitempty"  json:"provider,omitempty"`
	Recording string       `structs:"recording,omitempty" json:"recording,omitempty"`
	Match     *LyricsMatch `structs:"match,omitempty"     json:"match,omitempty"`
	Reason    string       `structs:"reason,omitempty"    json:"reason,omitempty"`
}

const (
	MaxLyricsEvidence          = 32
	MaxLyricsMatchReasons      = 8
	MaxLyricsMatchReasonBytes  = 160
	MaxLyricsEvidenceKindBytes = 32
)

// Valid reports whether the match uses a known status and stays bounded.
func (match LyricsMatch) Valid() bool {
	switch match.Status {
	case LyricsMatchUnassessed, LyricsMatchValidated, LyricsMatchAmbiguous, LyricsMatchMismatch:
	default:
		return false
	}
	if len(match.Reasons) > MaxLyricsMatchReasons || len(match.Recording) > MaxLyricsVariantRecordingBytes {
		return false
	}
	for _, reason := range match.Reasons {
		if len(reason) > MaxLyricsMatchReasonBytes {
			return false
		}
	}
	return true
}

// Valid reports whether the evidence kind and attribution are bounded.
func (evidence LyricsEvidence) Valid() bool {
	if evidence.Kind == "" || len(evidence.Kind) > MaxLyricsEvidenceKindBytes {
		return false
	}
	if len(evidence.Provider) > 128 || len(evidence.Recording) > MaxLyricsVariantRecordingBytes {
		return false
	}
	return evidence.Match == nil || evidence.Match.Valid()
}

// LyricsVariant is one listing row. PlaybackIndex is advisory: the identity is
// the stable link to the playback entry.
type LyricsVariant struct {
	PluginID      string                `json:"pluginId"`
	PluginName    string                `json:"pluginName,omitempty"`
	Identity      LyricsVariantIdentity `json:"identity"`
	Selection     string                `json:"selection"`
	Provider      string                `json:"provider,omitempty"`
	Language      string                `json:"language,omitempty"`
	Role          string                `json:"role,omitempty"`
	Format        string                `json:"format,omitempty"`
	PlaybackIndex int                   `json:"playbackIndex"`
	Original      LyricsOriginalInfo    `json:"original"`
	Match         *LyricsMatch          `json:"match,omitempty"`
}

// LyricsOriginal is one exact native document returned by a selection.
type LyricsOriginal struct {
	Identity   LyricsVariantIdentity `json:"identity"`
	Format     string                `json:"format"`
	Provenance string                `json:"provenance,omitempty"`
	MediaType  string                `json:"mediaType"`
	Extension  string                `json:"extension"`
	Document   []byte                `json:"document,omitempty"`
}

type lyricsSelection struct {
	Plugin   string                `json:"plugin"`
	Identity LyricsVariantIdentity `json:"identity"`
}

// EncodeLyricsSelection builds the bounded opaque token clients pass back to
// retrieve one exact selection. The token is not trusted: the identity is
// revalidated and must match a currently cached candidate.
func EncodeLyricsSelection(plugin string, identity LyricsVariantIdentity) (string, bool) {
	plugin = strings.TrimSpace(plugin)
	if !validPluginID(plugin) || !identity.Valid() {
		return "", false
	}
	payload, err := json.Marshal(lyricsSelection{Plugin: plugin, Identity: identity})
	if err != nil || len(payload) > MaxLyricsSelectionBytes {
		return "", false
	}
	return base64.RawURLEncoding.EncodeToString(payload), true
}

// DecodeLyricsSelection parses and validates an opaque selection token.
func DecodeLyricsSelection(token string) (string, LyricsVariantIdentity, bool) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > MaxLyricsSelectionBytes {
		return "", LyricsVariantIdentity{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(payload) > MaxLyricsSelectionBytes {
		return "", LyricsVariantIdentity{}, false
	}
	var selection lyricsSelection
	if err := json.Unmarshal(payload, &selection); err != nil {
		return "", LyricsVariantIdentity{}, false
	}
	if !validPluginID(selection.Plugin) || !selection.Identity.Valid() {
		return "", LyricsVariantIdentity{}, false
	}
	return selection.Plugin, selection.Identity, true
}

func validPluginID(plugin string) bool {
	if plugin == "" || len(plugin) > MaxLyricsVariantPluginBytes {
		return false
	}
	for _, char := range plugin {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '-', char == '_', char == '.':
		default:
			return false
		}
	}
	return true
}

// LyricsOriginalMediaType is the trusted media type for a native format.
func LyricsOriginalMediaType(format string) string {
	mediaType, _ := lyricsOriginalExtension(format)
	return mediaType
}

// LyricsOriginalExtension is the trusted download extension for a native format.
func LyricsOriginalExtension(format string) string {
	_, extension := lyricsOriginalExtension(format)
	return extension
}

// lyricsOriginalExtension maps validated source formats to safe headers. A
// plugin-supplied media type is never copied into a response.
func lyricsOriginalExtension(format string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "ttml":
		return "application/ttml+xml", ".ttml"
	case "lrc":
		return "text/plain; charset=utf-8", ".lrc"
	case "elrc":
		return "text/plain; charset=utf-8", ".lrc"
	case "qrc":
		return "application/xml", ".qrc"
	case "krc":
		return "text/plain; charset=utf-8", ".krc"
	case "yrc":
		return "text/plain; charset=utf-8", ".yrc"
	case "lyl":
		return "text/plain; charset=utf-8", ".lyl"
	case "wsy":
		return "application/octet-stream", ".wsy"
	case "lyricsfile":
		return "application/yaml", ".yaml"
	case "plain":
		return "text/plain; charset=utf-8", ".txt"
	}
	return "", ""
}
