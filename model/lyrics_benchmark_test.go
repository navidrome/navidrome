package model

import (
	"os"
	"path/filepath"
	"testing"
)

// Benchmark payloads are real public-domain lyrics ("Auld Lang Syne", Robert
// Burns, 1788) rendered into every supported format, so the numbers reflect
// realistic content and sizing. The same song across formats makes per-format
// cost directly comparable. Fixtures live in tests/fixtures/lyrics/.
func loadLyricFixture(b *testing.B, name string) []byte {
	b.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", "lyrics", name))
	if err != nil {
		b.Fatal(err)
	}
	return contents
}

func benchmarkParse(b *testing.B, suffix, fixture string) {
	contents := loadLyricFixture(b, fixture)
	b.ReportAllocs()
	b.SetBytes(int64(len(contents)))
	for b.Loop() {
		if _, err := ParseLyrics(b.Context(), suffix, "eng", contents); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLyrics_LRC(b *testing.B)         { benchmarkParse(b, ".lrc", "auld-lang-syne.lrc") }
func BenchmarkParseLyrics_Plain(b *testing.B)       { benchmarkParse(b, ".txt", "auld-lang-syne.txt") }
func BenchmarkParseLyrics_EnhancedLRC(b *testing.B) { benchmarkParse(b, ".lrc", "auld-lang-syne.elrc") }
func BenchmarkParseLyrics_SRT(b *testing.B)         { benchmarkParse(b, ".srt", "auld-lang-syne.srt") }
func BenchmarkParseLyrics_TTML(b *testing.B)        { benchmarkParse(b, ".ttml", "auld-lang-syne.ttml") }
func BenchmarkParseLyrics_YAML(b *testing.B)        { benchmarkParse(b, ".yaml", "auld-lang-syne.yaml") }

// Content-sniff path (empty suffix) — what embedded tags and plugins hit.
func BenchmarkParseLyrics_SniffTTML(b *testing.B)  { benchmarkParse(b, "", "auld-lang-syne.ttml") }
func BenchmarkParseLyrics_SniffSRT(b *testing.B)   { benchmarkParse(b, "", "auld-lang-syne.srt") }
func BenchmarkParseLyrics_SniffYAML(b *testing.B)  { benchmarkParse(b, "", "auld-lang-syne.yaml") }
func BenchmarkParseLyrics_SniffLRC(b *testing.B)   { benchmarkParse(b, "", "auld-lang-syne.lrc") }
func BenchmarkParseLyrics_SniffPlain(b *testing.B) { benchmarkParse(b, "", "auld-lang-syne.txt") }
