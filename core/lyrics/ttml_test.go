package lyrics

import (
	"testing"

	"github.com/navidrome/navidrome/model"
)

func TestParseTTML_MultiLanguageAndTiming(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:frameRate="30" ttp:subFrameRate="2" ttp:tickRate="10">
  <body>
    <div xml:lang="eng" begin="1s">
      <p begin="2s">Line one</p>
      <p begin="00:00:04:15.1"><span>Line two</span><br/>with break</p>
    </div>
    <div xml:lang="por">
      <p begin="45t">Linha</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 lyric tracks, got %d", len(list))
	}

	eng := list[0]
	if eng.Lang != "eng" {
		t.Fatalf("expected first track language 'eng', got %q", eng.Lang)
	}
	if !eng.Synced {
		t.Fatal("expected first track to be synced")
	}
	assertTimedLine(t, eng.Line[0], 3000, "Line one")
	assertTimedLine(t, eng.Line[1], 4517, "Line two\nwith break")

	por := list[1]
	if por.Lang != "por" {
		t.Fatalf("expected second track language 'por', got %q", por.Lang)
	}
	assertTimedLine(t, por.Line[0], 4500, "Linha")
}

func TestParseTTML_UnsupportedCueSkipped(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div>
      <p begin="wallclock(2026-01-01T00:00:00Z)">Skip me</p>
      <p begin="1s">Keep me</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(list))
	}
	if len(list[0].Line) != 1 {
		t.Fatalf("expected 1 line in lyric track, got %d", len(list[0].Line))
	}
	assertTimedLine(t, list[0].Line[0], 1000, "Keep me")
}

func TestParseTTML_BeginEndDurWithInheritance(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng" begin="10s">
    <div begin="5s" dur="8s">
      <p begin="1s" dur="2s">First line</p>
      <p begin="3s" end="5s">Second line</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(list))
	}
	if list[0].Lang != "eng" {
		t.Fatalf("expected language 'eng', got %q", list[0].Lang)
	}
	if len(list[0].Line) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(list[0].Line))
	}
	assertTimedLine(t, list[0].Line[0], 16000, "First line")
	assertTimedLine(t, list[0].Line[1], 18000, "Second line")
}

func TestParseTTML_NonStandardBareSecondOffsets(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng" begin="10">
    <div>
      <p begin="0.170">First line</p>
      <p begin="3.710">Second line</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(list))
	}
	if len(list[0].Line) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(list[0].Line))
	}
	assertTimedLine(t, list[0].Line[0], 10170, "First line")
	assertTimedLine(t, list[0].Line[1], 13710, "Second line")
}

func TestParseTTML_WordTimingTokens(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="00:01.000" end="00:03.000">
        <span begin="00:01.000" end="00:01.400">He</span><span begin="00:01.400" end="00:01.800">llo</span>
        <span ttm:role="x-bg"><span begin="00:02.000" end="00:02.500">echo</span></span>
      </p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(list))
	}
	if len(list[0].Line) != 1 {
		t.Fatalf("expected 1 line, got %d", len(list[0].Line))
	}

	line := list[0].Line[0]
	assertTimedLine(t, line, 1000, "Hello\necho")
	if line.End == nil || *line.End != 3000 {
		t.Fatalf("expected line end 3000, got %v", line.End)
	}
	if len(line.Token) != 3 {
		t.Fatalf("expected 3 timed tokens, got %d", len(line.Token))
	}

	assertToken(t, line.Token[0], 1000, 1400, "He", "")
	assertToken(t, line.Token[1], 1400, 1800, "llo", "")
	assertToken(t, line.Token[2], 2000, 2500, "echo", "x-bg")
}

func TestParseTTML_AmbiguousDecimalTimingPrefersAbsoluteWhenInsideParentWindow(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div begin="37.870" end="45.570">
      <p begin="43.444" end="45.570">
        <span begin="43.444" end="43.716">go</span>
        <span begin="43.716" end="43.887">go</span>
      </p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 || len(list[0].Line) != 1 {
		t.Fatalf("expected one parsed lyric line, got %#v", list)
	}

	line := list[0].Line[0]
	assertTimedLine(t, line, 43444, "go\ngo")
	if line.End == nil || *line.End != 45570 {
		t.Fatalf("expected line end 45570, got %v", line.End)
	}
	if len(line.Token) != 2 {
		t.Fatalf("expected 2 timed tokens, got %d", len(line.Token))
	}
	assertToken(t, line.Token[0], 43444, 43716, "go", "")
	assertToken(t, line.Token[1], 43716, 43887, "go", "")
}

func TestParseTTML_UnsyncedFallback(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body>
    <div>
      <p>No timing here</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 lyric track, got %d", len(list))
	}
	if list[0].Lang != "xxx" {
		t.Fatalf("expected default language 'xxx', got %q", list[0].Lang)
	}
	if list[0].Synced {
		t.Fatal("expected lyric track to be unsynced")
	}
	if len(list[0].Line) != 1 {
		t.Fatalf("expected 1 line, got %d", len(list[0].Line))
	}
	if list[0].Line[0].Start != nil {
		t.Fatalf("expected line start to be nil, got %v", *list[0].Line[0].Start)
	}
	if list[0].Line[0].Value != "No timing here" {
		t.Fatalf("expected line value %q, got %q", "No timing here", list[0].Line[0].Value)
	}
}

func TestParseTTML_MetadataTracksByKey(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <translations>
          <translation xml:lang="es">
            <text for="L1">Hola</text>
            <text for="MISSING">Skip me</text>
          </translation>
        </translations>
        <transliterations>
          <transliteration xml:lang="ja-Latn">
            <text for="L2"><span begin="00:02.000" end="00:02.300" xmlns="http://www.w3.org/ns/ttml">ko</span><span begin="00:02.300" end="00:02.600" xmlns="http://www.w3.org/ns/ttml">nni</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body xml:lang="ja">
    <div>
      <p begin="00:01.000" end="00:01.500" itunes:key="L1">こんにちは</p>
      <p begin="00:02.000" end="00:02.700" itunes:key="L2">こんばんは</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 lyric tracks, got %d", len(list))
	}

	main := list[0]
	if main.Kind != "main" {
		t.Fatalf("expected main track kind %q, got %q", "main", main.Kind)
	}
	if main.Lang != "ja" {
		t.Fatalf("expected main track language %q, got %q", "ja", main.Lang)
	}
	if len(main.Line) != 2 {
		t.Fatalf("expected 2 lines in main track, got %d", len(main.Line))
	}

	translation := list[1]
	if translation.Kind != "translation" {
		t.Fatalf("expected translation kind %q, got %q", "translation", translation.Kind)
	}
	if translation.Lang != "es" {
		t.Fatalf("expected translation language %q, got %q", "es", translation.Lang)
	}
	if len(translation.Line) != 1 {
		t.Fatalf("expected 1 translation line, got %d", len(translation.Line))
	}
	assertTimedLine(t, translation.Line[0], 1000, "Hola")
	if translation.Line[0].End == nil || *translation.Line[0].End != 1500 {
		t.Fatalf("expected translation line end %d, got %v", 1500, translation.Line[0].End)
	}

	pronunciation := list[2]
	if pronunciation.Kind != "pronunciation" {
		t.Fatalf("expected pronunciation kind %q, got %q", "pronunciation", pronunciation.Kind)
	}
	if pronunciation.Lang != "ja-latn" {
		t.Fatalf("expected pronunciation language %q, got %q", "ja-latn", pronunciation.Lang)
	}
	if len(pronunciation.Line) != 1 {
		t.Fatalf("expected 1 pronunciation line, got %d", len(pronunciation.Line))
	}
	assertTimedLine(t, pronunciation.Line[0], 2000, "konni")
	if pronunciation.Line[0].End == nil || *pronunciation.Line[0].End != 2600 {
		t.Fatalf("expected pronunciation line end %d, got %v", 2600, pronunciation.Line[0].End)
	}
	if len(pronunciation.Line[0].Token) != 2 {
		t.Fatalf("expected 2 pronunciation tokens, got %d", len(pronunciation.Line[0].Token))
	}
	assertToken(t, pronunciation.Line[0].Token[0], 2000, 2300, "ko", "")
	assertToken(t, pronunciation.Line[0].Token[1], 2300, 2600, "nni", "")
}

func TestParseTTML_PronunciationBareDecimalEndTimes(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <transliterations>
          <transliteration xml:lang="ja-Latn">
            <text for="L1"><span begin="2.747" end="3.018" xmlns="http://www.w3.org/ns/ttml">I</span> <span begin="3.018" end="3.179" xmlns="http://www.w3.org/ns/ttml">woke</span> <span begin="3.179" end="3.582" xmlns="http://www.w3.org/ns/ttml">up</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body xml:lang="ja">
    <div>
      <p begin="00:02.747" end="00:04.000" itunes:key="L1">起きた</p>
    </div>
  </body>
</tt>`)

	list, err := parseTTML(content)
	if err != nil {
		t.Fatalf("parseTTML returned error: %v", err)
	}

	var pronunciation *model.Lyrics
	for i := range list {
		if list[i].Kind == "pronunciation" {
			pronunciation = &list[i]
			break
		}
	}
	if pronunciation == nil {
		t.Fatal("expected a pronunciation track")
	}
	if len(pronunciation.Line) != 1 {
		t.Fatalf("expected 1 pronunciation line, got %d", len(pronunciation.Line))
	}

	line := pronunciation.Line[0]
	assertTimedLine(t, line, 2747, "I woke up")
	if len(line.Token) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(line.Token))
	}
	assertToken(t, line.Token[0], 2747, 3018, "I", "")
	assertToken(t, line.Token[1], 3018, 3179, "woke", "")
	assertToken(t, line.Token[2], 3179, 3582, "up", "")
}

func assertTimedLine(t *testing.T, line model.Line, expectedStart int64, expectedValue string) {
	t.Helper()

	if line.Start == nil {
		t.Fatal("expected line start to be set, got nil")
	}
	if *line.Start != expectedStart {
		t.Fatalf("expected line start %d, got %d", expectedStart, *line.Start)
	}
	if line.Value != expectedValue {
		t.Fatalf("expected line value %q, got %q", expectedValue, line.Value)
	}
}

func assertToken(t *testing.T, token model.Token, expectedStart int64, expectedEnd int64, expectedValue string, expectedRole string) {
	t.Helper()

	if token.Start == nil {
		t.Fatal("expected token start to be set, got nil")
	}
	if *token.Start != expectedStart {
		t.Fatalf("expected token start %d, got %d", expectedStart, *token.Start)
	}
	if token.End == nil {
		t.Fatal("expected token end to be set, got nil")
	}
	if *token.End != expectedEnd {
		t.Fatalf("expected token end %d, got %d", expectedEnd, *token.End)
	}
	if token.Value != expectedValue {
		t.Fatalf("expected token value %q, got %q", expectedValue, token.Value)
	}
	if token.Role != expectedRole {
		t.Fatalf("expected token role %q, got %q", expectedRole, token.Role)
	}
}
