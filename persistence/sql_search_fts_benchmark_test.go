package persistence

import (
	"context"
	"testing"
)

func BenchmarkSearchFTS5QueryCached(b *testing.B) {
	queries := []string{
		"beatles abbey road",
		`"taxman revolver"`,
		"rock AND jazz NOT pop",
		"café résumé",
		"嵐",
	}

	b.ReportAllocs()
	for b.Loop() {
		for _, query := range queries {
			_, _ = buildFTS5QueryCached(context.Background(), query)
		}
	}
}
