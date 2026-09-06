package lofty

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildRequestKeepsFilesInsideLibrary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := &extractor{baseDir: root}
	req, err := e.buildRequest(context.Background(), []string{"Artist/Album/01 Song.flac"})
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}
	if len(req.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(req.Files))
	}
	want := filepath.Join(root, "Artist", "Album", "01 Song.flac")
	if req.Files[0].Path != want {
		t.Fatalf("path = %q, want %q", req.Files[0].Path, want)
	}
}

func TestBuildRequestRejectsTraversal(t *testing.T) {
	t.Parallel()

	e := &extractor{baseDir: t.TempDir()}
	if _, err := e.buildRequest(context.Background(), []string{"../outside.flac"}); err == nil {
		t.Fatal("buildRequest() accepted path traversal")
	}
}

func TestConvertResponse(t *testing.T) {
	t.Parallel()

	modified := time.Date(2026, time.August, 26, 6, 30, 0, 123, time.UTC)
	created := modified.Add(-time.Hour)
	createdNS := created.UnixNano()
	resp := response{
		Protocol: protocolVersion,
		Results: map[string]rawResult{
			"song.m4a": {
				Tags:       map[string][]string{"title": {"Song"}},
				FileInfo:   &rawFileInfo{Name: "song.m4a", Size: 1234, ModifiedNS: modified.UnixNano(), CreatedNS: &createdNS},
				DurationNS: uint64((3*time.Minute + 12*time.Second).Nanoseconds()),
				BitRate:    256,
				BitDepth:   16,
				SampleRate: 44100,
				Channels:   2,
				Codec:      "alac",
				HasPicture: true,
			},
		},
	}
	got, err := convertResponse(resp)
	if err != nil {
		t.Fatalf("convertResponse() error = %v", err)
	}
	info := got["song.m4a"]
	if info.AudioProperties.Codec != "alac" {
		t.Fatalf("codec = %q, want alac", info.AudioProperties.Codec)
	}
	if info.AudioProperties.Duration != 3*time.Minute+12*time.Second {
		t.Fatalf("duration = %s", info.AudioProperties.Duration)
	}
	if !info.HasPicture {
		t.Fatal("HasPicture = false, want true")
	}
	if info.FileInfo == nil {
		t.Fatal("FileInfo = nil")
	}
	if info.FileInfo.Name() != "song.m4a" || info.FileInfo.Size() != 1234 {
		t.Fatalf("FileInfo = %q, %d", info.FileInfo.Name(), info.FileInfo.Size())
	}
	if !info.FileInfo.ModTime().Equal(modified) || !info.FileInfo.BirthTime().Equal(created) {
		t.Fatalf("file times = %s, %s", info.FileInfo.ModTime(), info.FileInfo.BirthTime())
	}
}

func TestConvertResponseAllowsOlderWorkerWithoutFileInfo(t *testing.T) {
	t.Parallel()

	got, err := convertResponse(response{Results: map[string]rawResult{"song.mp3": {}}})
	if err != nil {
		t.Fatalf("convertResponse() error = %v", err)
	}
	if got["song.mp3"].FileInfo != nil {
		t.Fatal("older worker result unexpectedly populated FileInfo")
	}
}

func TestConvertResponseReturnsRequestError(t *testing.T) {
	t.Parallel()

	_, err := convertResponse(response{Errors: map[string]string{"$request": "bad request"}})
	if err == nil || err.Error() != "bad request" {
		t.Fatalf("error = %v, want bad request", err)
	}
}
