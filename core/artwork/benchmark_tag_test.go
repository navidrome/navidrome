package artwork

import (
	"testing"

	"go.senan.xyz/taglib"
)

func BenchmarkTagExtraction(b *testing.B) {
	// Use existing test fixture with embedded artwork
	testFile := "tests/fixtures/artist/an-album/test.mp3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := taglib.OpenReadOnly(testFile, taglib.WithReadStyle(taglib.ReadStyleFast))
		if err != nil {
			b.Fatal(err)
		}
		images := f.Properties().Images
		if len(images) == 0 {
			b.Fatal("no images found in test file")
		}
		data, err := f.Image(0)
		if err != nil || len(data) == 0 {
			b.Fatal("failed to extract image data")
		}
		f.Close()
	}
}
