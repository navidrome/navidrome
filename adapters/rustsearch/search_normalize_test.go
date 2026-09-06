package rustsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/model"
)

func TestMediaFileDocumentUsesSearchNormalizedWhenPresent(t *testing.T) {
	t.Parallel()
	engine := New()
	mf := model.MediaFile{
		ID: "s1", Title: "a-ha", Album: "Hunting", Artist: "a-ha", AlbumArtist: "a-ha",
		LibraryID: 1, SearchNormalized: "aha hunting",
	}
	doc := engine.mediaFileDocument(context.Background(), mf)
	if !strings.Contains(doc.Secondary, "aha hunting") {
		t.Fatalf("secondary = %q, want search_normalized token", doc.Secondary)
	}
}

func TestMediaFileDocumentOmitsEmptySearchNormalized(t *testing.T) {
	t.Parallel()
	engine := New()
	mf := model.MediaFile{
		ID: "s2", Title: "R.E.M.", Album: "Automatic", Artist: "R.E.M.", AlbumArtist: "R.E.M.",
		LibraryID: 1,
	}
	doc := engine.mediaFileDocument(context.Background(), mf)
	// Go no longer calls metadata NormalizeMany; Rust Apply enrich covers FTS variants.
	if strings.Contains(doc.Secondary, "REM") && !strings.Contains(doc.Secondary, "R.E.M.") {
		t.Fatalf("unexpected pre-normalized-only secondary: %q", doc.Secondary)
	}
	if doc.Primary != "R.E.M." {
		t.Fatalf("primary = %q, want R.E.M.", doc.Primary)
	}
	if !strings.Contains(doc.Secondary, "Automatic") {
		t.Fatalf("secondary = %q, want raw album", doc.Secondary)
	}
}

func TestAlbumDocumentKeepsRawSecondaryWithoutEnsureHop(t *testing.T) {
	t.Parallel()
	engine := New()
	album := model.Album{ID: "a1", Name: "Need", AlbumArtist: "Norm", LibraryID: 1}
	doc := engine.albumDocument(context.Background(), album)
	if doc.Primary == "" {
		t.Fatal("expected primary")
	}
	if !strings.Contains(doc.Secondary, "Norm") {
		t.Fatalf("secondary = %q, want album artist", doc.Secondary)
	}
}
