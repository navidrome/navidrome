package lyrics

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/navidrome/navidrome/model"
)

func TestFromExternalFileTTML(t *testing.T) {
	ctx := context.Background()
	mf := model.MediaFile{Path: fixturePath("test.mp3")}

	lyrics, err := fromExternalFile(ctx, &mf, ".ttml")
	if err != nil {
		t.Fatalf("fromExternalFile returned error: %v", err)
	}
	if len(lyrics) != 2 {
		t.Fatalf("expected 2 lyric tracks, got %d", len(lyrics))
	}
	if lyrics[0].Lang != "eng" {
		t.Fatalf("expected first language 'eng', got %q", lyrics[0].Lang)
	}
	if len(lyrics[0].Line) != 2 {
		t.Fatalf("expected 2 english lines, got %d", len(lyrics[0].Line))
	}
	if lyrics[0].Line[0].Start == nil || *lyrics[0].Line[0].Start != 18800 {
		t.Fatalf("expected first english line start to be 18800, got %v", lyrics[0].Line[0].Start)
	}
}

func TestFromExternalFileTTMLWithUTF8BOM(t *testing.T) {
	ctx := context.Background()
	mf := model.MediaFile{Path: fixturePath("bom-test.ttml")}

	lyrics, err := fromExternalFile(ctx, &mf, ".ttml")
	if err != nil {
		t.Fatalf("fromExternalFile returned error: %v", err)
	}
	if len(lyrics) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(lyrics))
	}
	if !lyrics[0].Synced {
		t.Fatal("expected BOM TTML lyrics to be synced")
	}
	if len(lyrics[0].Line) != 1 {
		t.Fatalf("expected 1 lyric line, got %d", len(lyrics[0].Line))
	}
	if lyrics[0].Line[0].Start == nil || *lyrics[0].Line[0].Start != 0 {
		t.Fatalf("expected first line start 0, got %v", lyrics[0].Line[0].Start)
	}
}

func TestFromExternalFileTTMLUTF16(t *testing.T) {
	ctx := context.Background()
	mf := model.MediaFile{Path: fixturePath("bom-utf16-test.ttml")}

	lyrics, err := fromExternalFile(ctx, &mf, ".ttml")
	if err != nil {
		t.Fatalf("fromExternalFile returned error: %v", err)
	}
	if len(lyrics) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(lyrics))
	}
	if !lyrics[0].Synced {
		t.Fatal("expected UTF16 TTML lyrics to be synced")
	}
	if len(lyrics[0].Line) != 2 {
		t.Fatalf("expected 2 lyric lines, got %d", len(lyrics[0].Line))
	}
	if lyrics[0].Line[0].Start == nil || *lyrics[0].Line[0].Start != 18800 {
		t.Fatalf("expected first line start 18800, got %v", lyrics[0].Line[0].Start)
	}
	if lyrics[0].Line[1].Start == nil || *lyrics[0].Line[1].Start != 22801 {
		t.Fatalf("expected second line start 22801, got %v", lyrics[0].Line[1].Start)
	}
}

func fixturePath(name string) string {
	candidates := []string{
		filepath.Join("tests", "fixtures", name),
		filepath.Join("..", "..", "tests", "fixtures", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("tests", "fixtures", name)
}
