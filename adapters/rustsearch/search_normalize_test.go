package rustsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/model"
)

func TestMediaFileDocumentUsesSearchNormalizedWithoutReNormalize(t *testing.T) {
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

func TestEnsureAlbumSearchNormalizedSkipsPopulated(t *testing.T) {
	t.Parallel()
	albums := []model.Album{
		{ID: "a1", Name: "Already", AlbumArtist: "Set", SearchNormalized: "already set"},
		{ID: "a2", Name: "Need", AlbumArtist: "Norm"},
	}
	// Without a metadata worker, NormalizeMany returns empty for misses; populated stays.
	ensureAlbumSearchNormalized(context.Background(), albums)
	if albums[0].SearchNormalized != "already set" {
		t.Fatalf("populated SearchNormalized changed: %q", albums[0].SearchNormalized)
	}
}
