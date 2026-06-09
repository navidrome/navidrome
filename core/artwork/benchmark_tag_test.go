package artwork

import (
	"path/filepath"
	"runtime"
	"testing"

	"go.senan.xyz/taglib"
)

func BenchmarkTagExtraction(b *testing.B) {
	// Ensure working directory is the project root (tests.Init not called with -run='^$')
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller failed")
	}
	appPath, _ := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))

	// Use existing test fixture with embedded artwork
	testFile := filepath.Join(appPath, "tests/fixtures/artist/an-album/test.mp3")

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
