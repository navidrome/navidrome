package model

import (
	"strings"
	"testing"
)

func TestLyricsLocalIdentityIsStableAndOpaque(t *testing.T) {
	t.Parallel()

	first := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, "lrc", "en", "", ".lrc")
	second := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, "lrc", "fr", "translation", ".lrc")
	if !first.Valid() || !second.Valid() {
		t.Fatalf("expected valid identities: %+v %+v", first, second)
	}
	if first.Token != second.Token {
		t.Fatalf("same local source must keep one stable token: %q != %q", first.Token, second.Token)
	}
	if !strings.HasPrefix(first.Token, "local:v1:") {
		t.Fatalf("token %q is not host-issued", first.Token)
	}
	if strings.Contains(first.Token, "track-1") || strings.Contains(first.Token, ".lrc") || strings.Contains(first.Token, "/") {
		t.Fatalf("token must not expose the track, source or a path: %q", first.Token)
	}
	if first.SameAddress(second) == false {
		t.Fatal("same local source must compare by address")
	}
}

func TestLyricsLocalIdentityDiscriminatesSelectableVariants(t *testing.T) {
	t.Parallel()

	embedded := NewLyricsLocalIdentity("track-1", LyricsLocalKindEmbedded, "", "en", "", "embedded")
	lrc := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, "lrc", "en", "", ".lrc")
	elrc := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, "elrc", "en", "", ".elrc")
	otherTrack := NewLyricsLocalIdentity("track-2", LyricsLocalKindSidecar, "lrc", "en", "", ".lrc")

	seen := map[string]bool{}
	for _, identity := range []LyricsLocalIdentity{embedded, lrc, elrc, otherTrack} {
		if !identity.Valid() {
			t.Fatalf("invalid identity: %+v", identity)
		}
		if seen[identity.Token] {
			t.Fatalf("selectable local variants collided on %q", identity.Token)
		}
		seen[identity.Token] = true
	}
	if embedded.Kind != LyricsLocalKindEmbedded || lrc.Kind != LyricsLocalKindSidecar || lrc.Format != "lrc" {
		t.Fatalf("identity must distinguish kind and format: %+v %+v", embedded, lrc)
	}
}

func TestLyricsLocalIdentityRejectsUnboundedValues(t *testing.T) {
	t.Parallel()

	identity := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, strings.Repeat("x", 40), "en", "", ".x")
	if identity.Valid() {
		t.Fatal("oversized format must be invalid")
	}
	identity = LyricsLocalIdentity{Version: 1, Token: "local:v1:abcd", Kind: LyricsLocalKindSidecar, Format: "lrc"}
	if identity.Valid() {
		t.Fatal("short token must be invalid")
	}
	identity = LyricsLocalIdentity{Version: 2, Token: "local:v1:0123456789abcdef0123456789abcdef", Kind: LyricsLocalKindEmbedded}
	if identity.Valid() {
		t.Fatal("unknown version must be invalid")
	}
}

func TestRecordingScopeOnlyStripsContentRevision(t *testing.T) {
	t.Parallel()

	cases := []struct {
		recording string
		scope     string
		stable    bool
	}{
		{"1250@0123456789abcdef0123456789abcdef", "1250", true},
		{"id:0123456789abcdef@0123456789abcdef0123456789abcdef", "id:0123456789abcdef", true},
		{"provider@id@0123456789abcdef0123456789abcdef", "provider@id", true},
		{"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "", false},
		{"1250@XYZ", "", false},
		{"1250@0123", "", false},
		{"", "", false},
	}
	for _, test := range cases {
		scope, stable := (LyricsVariantIdentity{Recording: test.recording}).RecordingScope()
		if scope != test.scope || stable != test.stable {
			t.Fatalf("RecordingScope(%q) = %q, %v; want %q, %v", test.recording, scope, stable, test.scope, test.stable)
		}
	}
}

func TestLyricsPreferenceRoundTripAndTolerantDecode(t *testing.T) {
	t.Parallel()

	identity := LyricsVariantIdentity{
		Version: 1, Instance: "lrclib:0123456789abcdef0123456789abcdef",
		Recording: "1250@0123456789abcdef0123456789abcdef", Language: "en", Role: "main", Format: "lrc",
	}
	local := NewLyricsLocalIdentity("track-1", LyricsLocalKindSidecar, "lrc", "en", "", ".lrc")
	record := LyricsPreferenceRecord{
		Version:  LyricsPreferenceVersion,
		Revision: 7,
		Selection: &LyricsPreferenceSelection{
			Kind: LyricsPreferenceKindProvider, Plugin: "nd-better-lyrics", Provider: "LRCLib", Identity: &identity,
		},
		Offsets: []LyricsPreferenceOffset{
			{Kind: LyricsPreferenceKindProvider, Plugin: "nd-better-lyrics", Instance: identity.Instance, Recording: "1250", Language: "en", Role: "main", Format: "lrc", Stable: true, OffsetMs: 500},
			{Kind: LyricsPreferenceKindLocal, Local: &local, OffsetMs: -250},
		},
		Rejections: []LyricsPreferenceRejected{{
			Plugin: "nd-better-lyrics", Provider: "LRCLib", Instance: identity.Instance, Recording: "1250",
		}},
	}
	encoded, ok := EncodeLyricsPreference(record)
	if !ok {
		t.Fatal("expected a bounded record")
	}
	decoded, status := DecodeLyricsPreference(encoded)
	if status != LyricsPreferenceOK || decoded.Revision != 7 || len(decoded.Offsets) != 2 || len(decoded.Rejections) != 1 {
		t.Fatalf("round trip lost state: %+v status=%v", decoded, status)
	}
	if decoded.Selection == nil || decoded.Selection.Identity == nil || *decoded.Selection.Identity != identity {
		t.Fatalf("selection not preserved: %+v", decoded.Selection)
	}

	// Version 0 is earlier compatible data and is upgraded in memory.
	legacy := `{"version":0,"selection":{"kind":"local","local":{"version":1,"token":"` + local.Token + `","kind":"sidecar","format":"lrc"}},"offsets":[{"kind":"local","local":{"version":1,"token":"` + local.Token + `","kind":"sidecar","format":"lrc"},"offsetMs":150}]}`
	decoded, status = DecodeLyricsPreference(legacy)
	if status != LyricsPreferenceOK || decoded.Version != LyricsPreferenceVersion || decoded.Selection == nil || len(decoded.Offsets) != 1 {
		t.Fatalf("version 0 was not upgraded: %+v status=%v", decoded, status)
	}

	// Malformed entries are dropped without losing the record's other fields.
	partial := `{"version":1,"revision":3,"selection":{"kind":"provider","plugin":"","identity":null},"rejections":[{"plugin":"nd-better-lyrics","instance":"lrclib:x","recording":"1250"}]}`
	decoded, status = DecodeLyricsPreference(partial)
	if status != LyricsPreferenceOK || decoded.Selection != nil || len(decoded.Rejections) != 1 {
		t.Fatalf("invalid entries must be dropped: %+v status=%v", decoded, status)
	}
}

func TestLyricsPreferenceUnknownVersionIsNotOverwritten(t *testing.T) {
	t.Parallel()

	raw := `{"version":99,"revision":4,"selection":{"kind":"local"}}`
	record, status := DecodeLyricsPreference(raw)
	if status != LyricsPreferenceUnsupported {
		t.Fatalf("unknown version must be unsupported, got %v", status)
	}
	if record.Version != LyricsPreferenceVersion || record.Selection != nil {
		t.Fatalf("unsupported record must not be applied: %+v", record)
	}
	_, status = DecodeLyricsPreference(strings.Repeat("x", MaxLyricsPreferenceBytes+1))
	if status != LyricsPreferenceUnsupported {
		t.Fatalf("oversized record must be unsupported, got %v", status)
	}
	_, status = DecodeLyricsPreference("{not json")
	if status != LyricsPreferenceCorrupt {
		t.Fatalf("malformed record must be corrupt, got %v", status)
	}
}

func TestLyricsRejectionTokenRoundTrip(t *testing.T) {
	t.Parallel()

	token, ok := EncodeLyricsRejection("nd-better-lyrics", "lrclib:0123456789abcdef0123456789abcdef", "1250")
	if !ok {
		t.Fatal("expected a bounded rejection token")
	}
	plugin, instance, recording, ok := DecodeLyricsRejection(token)
	if !ok || plugin != "nd-better-lyrics" || instance != "lrclib:0123456789abcdef0123456789abcdef" || recording != "1250" {
		t.Fatalf("rejection token did not round trip: %q %q %q %v", plugin, instance, recording, ok)
	}
	for _, invalid := range []string{"", "not-base64", token + "x"} {
		if _, _, _, ok := DecodeLyricsRejection(invalid); ok {
			t.Fatalf("invalid token accepted: %q", invalid)
		}
	}
}
