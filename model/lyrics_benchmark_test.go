package model

import (
	"fmt"
	"strings"
	"testing"
)

// repeatLines builds a payload of n lines using the given per-line template,
// where %d is the line index, so benchmark inputs are realistically sized.
func repeatLines(n int, format string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, format, i)
	}
	return sb.String()
}

const benchLineCount = 200

func benchLRC() []byte {
	return []byte(repeatLines(benchLineCount, "[00:%02d.00]lyric line number with some words\n"))
}

func benchPlain() []byte {
	return []byte(repeatLines(benchLineCount, "lyric line number %d with some words\n"))
}

func benchEnhancedLRC() []byte {
	return []byte(repeatLines(benchLineCount, "[00:%02d.00]<00:00.10>word <00:00.50>by <00:00.90>word\n"))
}

func benchSRT() []byte {
	return []byte(repeatLines(benchLineCount, "%d\n00:00:01,000 --> 00:00:02,000\nsubtitle line\n\n"))
}

func benchTTML() []byte {
	var sb strings.Builder
	sb.WriteString(`<tt xmlns="http://www.w3.org/ns/ttml"><body><div>`)
	for i := 0; i < benchLineCount; i++ {
		fmt.Fprintf(&sb, `<p begin="00:00:%02d.000" end="00:00:%02d.000">ttml line</p>`, i%60, (i+1)%60)
	}
	sb.WriteString(`</div></body></tt>`)
	return []byte(sb.String())
}

func benchYAML() []byte {
	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nmetadata:\n  language: eng\nlines:\n")
	for i := 0; i < benchLineCount; i++ {
		fmt.Fprintf(&sb, "  - text: yaml line %d\n    start_ms: %d\n", i, i*1000)
	}
	return []byte(sb.String())
}

func benchmarkParse(b *testing.B, suffix string, contents []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(contents)))
	for b.Loop() {
		if _, err := ParseLyrics(suffix, "eng", contents); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLyrics_LRC(b *testing.B)         { benchmarkParse(b, ".lrc", benchLRC()) }
func BenchmarkParseLyrics_Plain(b *testing.B)       { benchmarkParse(b, ".txt", benchPlain()) }
func BenchmarkParseLyrics_EnhancedLRC(b *testing.B) { benchmarkParse(b, ".lrc", benchEnhancedLRC()) }
func BenchmarkParseLyrics_SRT(b *testing.B)         { benchmarkParse(b, ".srt", benchSRT()) }
func BenchmarkParseLyrics_TTML(b *testing.B)        { benchmarkParse(b, ".ttml", benchTTML()) }
func BenchmarkParseLyrics_YAML(b *testing.B)        { benchmarkParse(b, ".yaml", benchYAML()) }

// Content-sniff path (empty suffix) — what embedded tags and plugins hit.
func BenchmarkParseLyrics_SniffTTML(b *testing.B)  { benchmarkParse(b, "", benchTTML()) }
func BenchmarkParseLyrics_SniffSRT(b *testing.B)   { benchmarkParse(b, "", benchSRT()) }
func BenchmarkParseLyrics_SniffYAML(b *testing.B)  { benchmarkParse(b, "", benchYAML()) }
func BenchmarkParseLyrics_SniffLRC(b *testing.B)   { benchmarkParse(b, "", benchLRC()) }
func BenchmarkParseLyrics_SniffPlain(b *testing.B) { benchmarkParse(b, "", benchPlain()) }
