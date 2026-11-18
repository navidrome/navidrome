package lyrics

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

func embeddedMediaFile(t *testing.T, id string) *model.MediaFile {
	t.Helper()
	list := model.LyricList{{
		Kind: model.LyricKindMain, Lang: "en", Synced: true,
		Line: []model.Line{{Value: "embedded line", Start: new(int64(1000))}},
	}}
	raw, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	return &model.MediaFile{ID: id, Lyrics: string(raw)}
}

func TestEmbeddedVariantGetsStableHostIdentity(t *testing.T) {
	ctx := context.Background()
	mf := embeddedMediaFile(t, "track-embedded")
	first, err := fromEmbedded(ctx, mf)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fromEmbedded(ctx, mf)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Local == nil {
		t.Fatalf("embedded lyrics did not get a local identity: %+v", first)
	}
	if first[0].Selection != "" || first[0].Variant != nil {
		t.Fatalf("local identity must stay separate from provider selections: %+v", first[0])
	}
	if first[0].Local.Token != second[0].Local.Token {
		t.Fatalf("embedded identity is not stable: %q != %q", first[0].Local.Token, second[0].Local.Token)
	}
	if first[0].Local.Kind != model.LyricsLocalKindEmbedded || first[0].Local.Format != "" {
		t.Fatalf("embedded identity fields are wrong: %+v", first[0].Local)
	}
	if first[0].Local.Token == "" || first[0].Local.Token == "local:v1:" {
		t.Fatalf("embedded token is empty: %+v", first[0].Local)
	}
}

func TestLocalVariantsDistinguishFormats(t *testing.T) {
	previous := conf.Server.LyricsPriority
	conf.Server.LyricsPriority = "embedded"
	t.Cleanup(func() { conf.Server.LyricsPriority = previous })

	mf := embeddedMediaFile(t, "track-sidecar")
	embedded := LocalVariants(context.Background(), mf)
	if len(embedded) != 1 || embedded[0].Kind != model.LyricsLocalKindEmbedded {
		t.Fatalf("embedded variant was not listed: %+v", embedded)
	}

	lrc := withLocalIdentity(mf, model.LyricsLocalKindSidecar, "lrc", ".lrc", model.LyricList{{
		Kind: model.LyricKindMain, Lang: "en", Line: []model.Line{{Value: "lrc line"}},
	}})
	ttml := withLocalIdentity(mf, model.LyricsLocalKindSidecar, "ttml", ".ttml", model.LyricList{{
		Kind: model.LyricKindMain, Lang: "en", Line: []model.Line{{Value: "ttml line"}},
	}})
	if lrc[0].Local == nil || ttml[0].Local == nil {
		t.Fatal("sidecar lyrics did not get host identities")
	}
	if lrc[0].Local.Kind != model.LyricsLocalKindSidecar || lrc[0].Local.Format != "lrc" {
		t.Fatalf("sidecar identity fields are wrong: %+v", lrc[0].Local)
	}
	if lrc[0].Local.Token == ttml[0].Local.Token {
		t.Fatal("sidecar formats collided on one identity")
	}
	if lrc[0].Selection != "" || lrc[0].Variant != nil {
		t.Fatal("local identity must stay separate from provider selections")
	}
	for _, identity := range []model.LyricsLocalIdentity{*lrc[0].Local, *ttml[0].Local} {
		if filepath.IsAbs(identity.Token) || len(identity.Token) > model.MaxLyricsLocalTokenBytes {
			t.Fatalf("identity leaks a path or is unbounded: %+v", identity)
		}
	}
}
