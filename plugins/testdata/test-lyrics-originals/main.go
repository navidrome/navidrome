// Test lyrics plugin with optional variant identities and exact native
// documents. The two variants share one provider so identity, not the upstream
// label, must keep them individually addressable.
package main

import (
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/lyrics"
	"github.com/navidrome/navidrome/plugins/pdk/go/lyricsvariants"
)

const (
	providerName = "Native Provider"
	qrcDocument  = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<QrcInfos>\n<QrcHeadInfo SaveTime=\"231\" Version=\"100\"/>\n<LyricInfo LyricCount=\"1\">\n<Lyric_1 LyricType=\"1\" LyricContent=\"[0,1000]line(0,500) (500,500)one\"/>\n</LyricInfo>\n</QrcInfos>\n"
	ttmlDocument = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<tt xmlns=\"http://www.w3.org/ns/ttml\"><body><div><p begin=\"00:00.00\" end=\"00:01.00\">line one</p></div></body></tt>"
	elrcPlayback = "[00:00.00]line one"
)

func main() {}

func init() {
	impl := &testLyricsOriginals{}
	lyrics.Register(impl)
	lyricsvariants.Register(impl)
}

type testLyricsOriginals struct{}

func (t *testLyricsOriginals) GetLyrics(lyrics.GetLyricsRequest) (lyrics.GetLyricsResponse, error) {
	return lyrics.GetLyricsResponse{
		Lyrics: []lyrics.LyricsText{{Text: elrcPlayback}},
		Source: &lyrics.LyricsSource{Provider: providerName, Format: "elrc"},
	}, nil
}

func (t *testLyricsOriginals) GetLyricsV2(lyricsvariants.GetLyricsV2Request) (lyricsvariants.GetLyricsV2Response, error) {
	return lyricsvariants.GetLyricsV2Response{
		Lyrics: []lyricsvariants.LyricsV2Text{
			{Text: elrcPlayback, Variant: qrcIdentity(), Match: validatedMatch()},
			{Text: ttmlDocument, Variant: ttmlIdentity(), Match: ambiguousMatch()},
		},
		Source:   &lyricsvariants.LyricsSource{Provider: providerName, Format: "elrc"},
		Evidence: []lyricsvariants.LyricsEvidence{{Kind: "instrumental", Provider: "LRCLib", Recording: "rec-instrumental", Match: validatedMatch(), Reason: "provider declared instrumental"}},
	}, nil
}

func (t *testLyricsOriginals) GetLyricsVariants(lyricsvariants.GetLyricsVariantsRequest) (lyricsvariants.GetLyricsVariantsResponse, error) {
	return lyricsvariants.GetLyricsVariantsResponse{
		Variants: []lyricsvariants.LyricsVariantEntry{
			{
				Identity: *qrcIdentity(), Provider: providerName, Role: "main", Format: "elrc", PlaybackIndex: 0,
				Original: lyricsvariants.LyricsOriginalInfo{Available: true, Format: "qrc", Provenance: "supplied", MediaType: "application/xml", Extension: ".qrc", Size: len(qrcDocument)},
				Match:    validatedMatch(),
			},
			{
				Identity: *ttmlIdentity(), Provider: providerName, Role: "main", Format: "ttml", PlaybackIndex: 1,
				Original: lyricsvariants.LyricsOriginalInfo{Available: true, Format: "ttml", Provenance: "supplied", MediaType: "application/ttml+xml", Extension: ".ttml", Size: len(ttmlDocument)},
				Match:    ambiguousMatch(),
			},
		},
		Evidence: []lyricsvariants.LyricsEvidence{{Kind: "instrumental", Provider: "LRCLib", Recording: "rec-instrumental", Match: validatedMatch(), Reason: "provider declared instrumental"}},
	}, nil
}

func (t *testLyricsOriginals) GetLyricsOriginal(request lyricsvariants.GetLyricsOriginalRequest) (lyricsvariants.GetLyricsOriginalResponse, error) {
	response := lyricsvariants.GetLyricsOriginalResponse{Identity: request.Identity}
	switch {
	case strings.HasSuffix(request.Identity.Recording, "qrc"):
		response.Available = true
		response.Format = "qrc"
		response.Provenance = "supplied"
		response.MediaType = "application/xml"
		response.Extension = ".qrc"
		response.Document = []byte(qrcDocument)
	case strings.HasSuffix(request.Identity.Recording, "ttml"):
		response.Available = true
		response.Format = "ttml"
		response.Provenance = "supplied"
		response.MediaType = "application/ttml+xml"
		response.Extension = ".ttml"
		response.Document = []byte(ttmlDocument)
	default:
		response.UnavailableReason = "selection no longer offered"
	}
	return response, nil
}

func validatedMatch() *lyricsvariants.LyricsMatch {
	return &lyricsvariants.LyricsMatch{Status: "validated", Reasons: []string{"duration-within-2s"}, AutoSaveEligible: true}
}

func ambiguousMatch() *lyricsvariants.LyricsMatch {
	return &lyricsvariants.LyricsMatch{Status: "ambiguous", Reasons: []string{"version-unconfirmed"}, AutoSaveEligible: false}
}

func qrcIdentity() *lyricsvariants.LyricsVariant {
	return &lyricsvariants.LyricsVariant{Version: 1, Instance: "native:0123456789abcdef0123456789abcdef", Recording: "rec-qrc", Role: "main", Format: "elrc"}
}

func ttmlIdentity() *lyricsvariants.LyricsVariant {
	return &lyricsvariants.LyricsVariant{Version: 1, Instance: "native:0123456789abcdef0123456789abcdef", Recording: "rec-ttml", Role: "main", Format: "ttml"}
}
