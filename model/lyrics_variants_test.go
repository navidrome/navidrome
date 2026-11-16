package model

import (
	"strings"
	"testing"
)

func TestLyricsVariantIdentityValidation(t *testing.T) {
	t.Parallel()

	valid := LyricsVariantIdentity{
		Version: 1, Instance: "lrclib:0123456789abcdef0123456789abcdef",
		Recording: "1250", Language: "en", Role: "main", Format: "lrc",
	}
	if !valid.Valid() {
		t.Fatalf("identity = %#v, want valid", valid)
	}

	cases := map[string]LyricsVariantIdentity{
		"version":      {Version: 2, Instance: valid.Instance, Recording: "r", Role: "main", Format: "lrc"},
		"instance":     {Version: 1, Recording: "r", Role: "main", Format: "lrc"},
		"recording":    {Version: 1, Instance: valid.Instance, Role: "main", Format: "lrc"},
		"format":       {Version: 1, Instance: valid.Instance, Recording: "r", Role: "main"},
		"unknown role": {Version: 1, Instance: valid.Instance, Recording: "r", Role: "chorus", Format: "lrc"},
		"non-playback": {Version: 1, Instance: valid.Instance, Recording: "r", Role: "main", Format: "qrc"},
		"oversized":    {Version: 1, Instance: strings.Repeat("a", MaxLyricsVariantInstanceBytes+1), Recording: "r", Role: "main", Format: "lrc"},
	}
	for name, identity := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if identity.Valid() {
				t.Fatalf("identity %#v must be rejected", identity)
			}
		})
	}
}

func TestLyricsSelectionRoundTripAndBounds(t *testing.T) {
	t.Parallel()

	identity := LyricsVariantIdentity{
		Version: 1, Instance: "better-lyrics:0123456789abcdef0123456789abcdef",
		Recording: "sha256:abcd", Role: "main", Format: "ttml",
	}
	token, ok := EncodeLyricsSelection("nd-better-lyrics", identity)
	if !ok || token == "" {
		t.Fatal("valid selection must encode")
	}
	plugin, decoded, ok := DecodeLyricsSelection(token)
	if !ok || plugin != "nd-better-lyrics" || decoded != identity {
		t.Fatalf("decoded = %q/%#v/%v, want the exact selection", plugin, decoded, ok)
	}

	if _, _, ok := DecodeLyricsSelection(token + "tamper"); ok {
		t.Fatal("tampered token must be rejected")
	}
	if _, _, ok := DecodeLyricsSelection(strings.Repeat("a", MaxLyricsSelectionBytes+1)); ok {
		t.Fatal("oversized token must be rejected")
	}
	if _, ok := EncodeLyricsSelection("../escape", identity); ok {
		t.Fatal("plugin ids must be bounded identifiers")
	}
	if _, ok := EncodeLyricsSelection("plugin", LyricsVariantIdentity{Version: 2, Instance: identity.Instance, Recording: "r", Role: "main", Format: "lrc"}); ok {
		t.Fatal("unsupported identity versions must not encode")
	}
}

func TestLyricsOriginalHeadersAreTrustedServerValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ mediaType, extension string }{
		"ttml":       {"application/ttml+xml", ".ttml"},
		"qrc":        {"application/xml", ".qrc"},
		"wsy":        {"application/octet-stream", ".wsy"},
		"lyricsfile": {"application/json", ".json"},
	}
	for format, want := range cases {
		if got := LyricsOriginalMediaType(format); got != want.mediaType {
			t.Fatalf("%s media type = %q, want %q", format, got, want.mediaType)
		}
		if got := LyricsOriginalExtension(format); got != want.extension {
			t.Fatalf("%s extension = %q, want %q", format, got, want.extension)
		}
	}
	if LyricsOriginalMediaType("bogus") != "" || LyricsOriginalExtension("bogus") != "" {
		t.Fatal("unknown formats must not produce headers")
	}
}
