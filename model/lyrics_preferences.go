package model

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// LyricsPreferenceVersion is the current durable personal-preference schema.
// Version 0 records are earlier compatible data: they are read as version 1
// and upgraded in place by the next deliberate write. Unknown later versions
// are never applied or overwritten.
const LyricsPreferenceVersion = 1

const (
	MaxLyricsPreferenceBytes       = 8 << 10
	MaxLyricsPreferenceOffsets     = 16
	MaxLyricsPreferenceRejections  = 16
	MaxLyricsPreferenceProviderLen = 128
	MaxLyricsRejectionTokenBytes   = 512
)

const (
	LyricsPreferenceKindProvider = "provider"
	LyricsPreferenceKindLocal    = "local"
)

// LyricsPreferenceStatus reports whether a stored record can be interpreted.
type LyricsPreferenceStatus int

const (
	LyricsPreferenceOK LyricsPreferenceStatus = iota
	LyricsPreferenceCorrupt
	LyricsPreferenceUnsupported
)

// LyricsPreferenceRecord is the durable per-account, per-track personal state.
// It is stored in user_props, never in the plugin KV or a shared cache.
type LyricsPreferenceRecord struct {
	Version    int                        `json:"version"`
	Revision   int64                      `json:"revision,omitempty"`
	Selection  *LyricsPreferenceSelection `json:"selection,omitempty"`
	Offsets    []LyricsPreferenceOffset   `json:"offsets,omitempty"`
	Rejections []LyricsPreferenceRejected `json:"rejections,omitempty"`
}

// LyricsPreferenceSelection is the explicitly pinned variant, either a
// provider recording variant or a host-issued local variant.
type LyricsPreferenceSelection struct {
	Kind     string                 `json:"kind"`
	Plugin   string                 `json:"plugin,omitempty"`
	Provider string                 `json:"provider,omitempty"`
	Identity *LyricsVariantIdentity `json:"identity,omitempty"`
	Local    *LyricsLocalIdentity   `json:"local,omitempty"`
}

// LyricsPreferenceOffset is one variant-specific timing correction. Provider
// offsets carry the stable provider recording scope; local offsets carry the
// host-issued local identity. Stable is false for content-only provider
// addresses, which must never be rebound after a document change.
type LyricsPreferenceOffset struct {
	Kind      string               `json:"kind"`
	Plugin    string               `json:"plugin,omitempty"`
	Instance  string               `json:"instance,omitempty"`
	Recording string               `json:"recording,omitempty"`
	Language  string               `json:"language,omitempty"`
	Role      string               `json:"role,omitempty"`
	Format    string               `json:"format,omitempty"`
	Stable    bool                 `json:"stable,omitempty"`
	OffsetMs  int64                `json:"offsetMs"`
	Local     *LyricsLocalIdentity `json:"local,omitempty"`
}

// LyricsPreferenceRejected is one provider recording the listener rejected.
// It is source-instance scoped and recording-wide: language, role, format and
// content revision deliberately do not participate.
type LyricsPreferenceRejected struct {
	Plugin    string `json:"plugin"`
	Provider  string `json:"provider,omitempty"`
	Instance  string `json:"instance"`
	Recording string `json:"recording"`
}

// RecordingScope returns the stable provider recording id shared by every
// content revision of one recording. The plugin appends a 16-byte content
// digest after '@'; an address that is only a content digest has no stable
// scope and must not be rebound.
func (identity LyricsVariantIdentity) RecordingScope() (string, bool) {
	recording := strings.TrimSpace(identity.Recording)
	at := strings.LastIndexByte(recording, '@')
	if at <= 0 || len(recording)-at-1 != 32 {
		return "", false
	}
	for _, char := range recording[at+1:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return "", false
		}
	}
	return recording[:at], true
}

// DecodeLyricsPreference parses a stored record leniently. Malformed entries
// are dropped, known older versions are upgraded in memory, and unknown later
// versions are reported unsupported so callers never overwrite them.
func DecodeLyricsPreference(raw string) (LyricsPreferenceRecord, LyricsPreferenceStatus) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return LyricsPreferenceRecord{Version: LyricsPreferenceVersion}, LyricsPreferenceOK
	}
	if len(raw) > MaxLyricsPreferenceBytes {
		return LyricsPreferenceRecord{Version: LyricsPreferenceVersion}, LyricsPreferenceUnsupported
	}
	var record LyricsPreferenceRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return LyricsPreferenceRecord{Version: LyricsPreferenceVersion}, LyricsPreferenceCorrupt
	}
	switch record.Version {
	case 0, LyricsPreferenceVersion:
	default:
		return LyricsPreferenceRecord{Version: LyricsPreferenceVersion}, LyricsPreferenceUnsupported
	}
	record.Version = LyricsPreferenceVersion
	if record.Revision < 0 {
		record.Revision = 0
	}
	record.Selection = validPreferenceSelection(record.Selection)
	record.Offsets = validPreferenceOffsets(record.Offsets)
	record.Rejections = validPreferenceRejections(record.Rejections)
	return record, LyricsPreferenceOK
}

func validPreferenceSelection(selection *LyricsPreferenceSelection) *LyricsPreferenceSelection {
	if selection == nil {
		return nil
	}
	selection.Plugin = strings.TrimSpace(selection.Plugin)
	selection.Provider = strings.TrimSpace(selection.Provider)
	if len(selection.Provider) > MaxLyricsPreferenceProviderLen {
		return nil
	}
	switch selection.Kind {
	case LyricsPreferenceKindProvider:
		if !validPluginID(selection.Plugin) || selection.Identity == nil || !selection.Identity.Valid() {
			return nil
		}
		selection.Local = nil
	case LyricsPreferenceKindLocal:
		selection.Local = normalizeLocalIdentity(selection.Local)
		if selection.Local == nil || !selection.Local.Valid() {
			return nil
		}
		selection.Plugin = ""
		selection.Identity = nil
	default:
		return nil
	}
	return selection
}

func validPreferenceOffsets(offsets []LyricsPreferenceOffset) []LyricsPreferenceOffset {
	result := make([]LyricsPreferenceOffset, 0, min(len(offsets), MaxLyricsPreferenceOffsets))
	seen := map[string]bool{}
	for _, offset := range offsets {
		if len(result) >= MaxLyricsPreferenceOffsets {
			break
		}
		offset.OffsetMs = ClampLyricsOffset(offset.OffsetMs)
		if offset.OffsetMs == 0 {
			continue
		}
		switch offset.Kind {
		case LyricsPreferenceKindProvider:
			offset.Plugin = strings.TrimSpace(offset.Plugin)
			offset.Instance = strings.TrimSpace(offset.Instance)
			offset.Recording = strings.TrimSpace(offset.Recording)
			offset.Language = strings.TrimSpace(offset.Language)
			offset.Role = strings.TrimSpace(offset.Role)
			offset.Format = strings.ToLower(strings.TrimSpace(offset.Format))
			offset.Local = nil
			if !validPluginID(offset.Plugin) || offset.Instance == "" || offset.Recording == "" || offset.Format == "" {
				continue
			}
			if len(offset.Instance) > MaxLyricsVariantInstanceBytes || len(offset.Recording) > MaxLyricsVariantRecordingBytes ||
				len(offset.Language) > MaxLyricsVariantLanguageBytes || len(offset.Role) > MaxLyricsVariantRoleBytes ||
				len(offset.Format) > MaxLyricsVariantFormatBytes {
				continue
			}
			switch offset.Role {
			case "", LyricKindMain, LyricKindTranslation, LyricKindPronunciation:
			default:
				continue
			}
		case LyricsPreferenceKindLocal:
			offset.Local = normalizeLocalIdentity(offset.Local)
			if offset.Local == nil || !offset.Local.Valid() {
				continue
			}
			offset.Plugin, offset.Instance, offset.Recording = "", "", ""
			offset.Language, offset.Role, offset.Format, offset.Stable = "", "", "", false
		default:
			continue
		}
		key := offset.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, offset)
	}
	return result
}

func validPreferenceRejections(rejections []LyricsPreferenceRejected) []LyricsPreferenceRejected {
	result := make([]LyricsPreferenceRejected, 0, min(len(rejections), MaxLyricsPreferenceRejections))
	seen := map[string]bool{}
	for _, rejection := range rejections {
		if len(result) >= MaxLyricsPreferenceRejections {
			break
		}
		rejection.Plugin = strings.TrimSpace(rejection.Plugin)
		rejection.Provider = strings.TrimSpace(rejection.Provider)
		rejection.Instance = strings.TrimSpace(rejection.Instance)
		rejection.Recording = strings.TrimSpace(rejection.Recording)
		if !validPluginID(rejection.Plugin) || rejection.Instance == "" || rejection.Recording == "" {
			continue
		}
		if len(rejection.Provider) > MaxLyricsPreferenceProviderLen ||
			len(rejection.Instance) > MaxLyricsVariantInstanceBytes ||
			len(rejection.Recording) > MaxLyricsVariantRecordingBytes {
			continue
		}
		key := rejection.Plugin + "\x00" + rejection.Instance + "\x00" + rejection.Recording
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, rejection)
	}
	return result
}

func normalizeLocalIdentity(local *LyricsLocalIdentity) *LyricsLocalIdentity {
	if local == nil {
		return nil
	}
	if strings.TrimSpace(local.Role) == "" {
		local.Role = LyricKindMain
	}
	return local
}

// MaxLyricsOffsetMs bounds one variant-specific timing correction.
const MaxLyricsOffsetMs = 5000

// ClampLyricsOffset bounds a correction to the client-supported range.
func ClampLyricsOffset(value int64) int64 {
	if value > MaxLyricsOffsetMs {
		return MaxLyricsOffsetMs
	}
	if value < -MaxLyricsOffsetMs {
		return -MaxLyricsOffsetMs
	}
	return value
}

// Key identifies one offset's variant scope for replacement and deduplication.
func (offset LyricsPreferenceOffset) Key() string {
	if offset.Kind == LyricsPreferenceKindLocal && offset.Local != nil {
		return "local\x00" + offset.Local.Token
	}
	return "provider\x00" + offset.Plugin + "\x00" + offset.Instance + "\x00" + offset.Recording +
		"\x00" + offset.Language + "\x00" + offset.Role + "\x00" + offset.Format
}

// EncodeLyricsPreference serializes a bounded record.
func EncodeLyricsPreference(record LyricsPreferenceRecord) (string, bool) {
	record.Version = LyricsPreferenceVersion
	if record.Revision < 0 {
		record.Revision = 0
	}
	payload, err := json.Marshal(record)
	if err != nil || len(payload) > MaxLyricsPreferenceBytes {
		return "", false
	}
	return string(payload), true
}

type lyricsRejectionToken struct {
	Plugin    string `json:"plugin"`
	Instance  string `json:"instance"`
	Recording string `json:"recording"`
}

// EncodeLyricsRejection builds the bounded opaque token a listener uses to
// restore a rejected provider recording. It is not an authorization token:
// the host always resolves it against the authenticated listener's track.
func EncodeLyricsRejection(plugin, instance, recording string) (string, bool) {
	plugin = strings.TrimSpace(plugin)
	instance = strings.TrimSpace(instance)
	recording = strings.TrimSpace(recording)
	if !validPluginID(plugin) || instance == "" || recording == "" ||
		len(instance) > MaxLyricsVariantInstanceBytes || len(recording) > MaxLyricsVariantRecordingBytes {
		return "", false
	}
	payload, err := json.Marshal(lyricsRejectionToken{Plugin: plugin, Instance: instance, Recording: recording})
	if err != nil || len(payload) > MaxLyricsRejectionTokenBytes {
		return "", false
	}
	return base64.RawURLEncoding.EncodeToString(payload), true
}

// DecodeLyricsRejection parses and validates a rejection token.
func DecodeLyricsRejection(token string) (string, string, string, bool) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > MaxLyricsRejectionTokenBytes {
		return "", "", "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(payload) > MaxLyricsRejectionTokenBytes {
		return "", "", "", false
	}
	var decoded lyricsRejectionToken
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", "", "", false
	}
	if !validPluginID(decoded.Plugin) || decoded.Instance == "" || decoded.Recording == "" ||
		len(decoded.Instance) > MaxLyricsVariantInstanceBytes || len(decoded.Recording) > MaxLyricsVariantRecordingBytes {
		return "", "", "", false
	}
	return decoded.Plugin, decoded.Instance, decoded.Recording, true
}
