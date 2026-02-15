package apimodel_test

import (
	"testing"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/nativeapi/apimodel"
)

func TestFromMediaFile(t *testing.T) {
	now := time.Now()
	playDate := now.Add(-time.Hour)
	starredAt := now.Add(-2 * time.Hour)
	rgGain := 1.5
	rgPeak := 0.9

	mf := model.MediaFile{
		ID:           "mf-1",
		LibraryID:    1,
		LibraryName:  "My Music",
		LibraryPath:  "/music",
		FolderID:     "folder-1",
		Path:         "/music/song.mp3",
		Title:        "Test Song",
		Album:        "Test Album",
		AlbumID:      "album-1",
		Artist:       "Test Artist",
		ArtistID:     "artist-1",
		AlbumArtist:  "Test Album Artist",
		Duration:     180.5,
		Size:         5242880,
		Suffix:       "mp3",
		BitRate:      320,
		SampleRate:   44100,
		BitDepth:     16,
		Channels:     2,
		Year:         2024,
		TrackNumber:  3,
		DiscNumber:   1,
		Genre:        "Rock",
		HasCoverArt:  true,
		RGAlbumGain:  &rgGain,
		RGAlbumPeak:  &rgPeak,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	mf.PlayCount = 5
	mf.PlayDate = &playDate
	mf.Starred = true
	mf.StarredAt = &starredAt
	mf.Rating = 4
	mf.BookmarkPosition = 1000

	song := apimodel.FromMediaFile(mf)

	// Verify mapped fields
	if song.ID != mf.ID {
		t.Errorf("ID: got %q, want %q", song.ID, mf.ID)
	}
	if song.Title != mf.Title {
		t.Errorf("Title: got %q, want %q", song.Title, mf.Title)
	}
	if song.Artist != mf.Artist {
		t.Errorf("Artist: got %q, want %q", song.Artist, mf.Artist)
	}
	if song.Album != mf.Album {
		t.Errorf("Album: got %q, want %q", song.Album, mf.Album)
	}
	if song.Duration != mf.Duration {
		t.Errorf("Duration: got %v, want %v", song.Duration, mf.Duration)
	}
	if song.BitRate != mf.BitRate {
		t.Errorf("BitRate: got %d, want %d", song.BitRate, mf.BitRate)
	}
	if song.LibraryID != mf.LibraryID {
		t.Errorf("LibraryID: got %d, want %d", song.LibraryID, mf.LibraryID)
	}
	if song.LibraryName != mf.LibraryName {
		t.Errorf("LibraryName: got %q, want %q", song.LibraryName, mf.LibraryName)
	}

	// Verify annotations are mapped
	if song.PlayCount != mf.PlayCount {
		t.Errorf("PlayCount: got %d, want %d", song.PlayCount, mf.PlayCount)
	}
	if song.Starred != mf.Starred {
		t.Errorf("Starred: got %v, want %v", song.Starred, mf.Starred)
	}
	if song.Rating != mf.Rating {
		t.Errorf("Rating: got %d, want %d", song.Rating, mf.Rating)
	}

	// Verify bookmark is mapped
	if song.BookmarkPosition != mf.BookmarkPosition {
		t.Errorf("BookmarkPosition: got %d, want %d", song.BookmarkPosition, mf.BookmarkPosition)
	}

	// Verify replay gain pointers are mapped
	if song.RGAlbumGain == nil || *song.RGAlbumGain != rgGain {
		t.Errorf("RGAlbumGain: got %v, want %v", song.RGAlbumGain, &rgGain)
	}
}

func TestFromMediaFiles(t *testing.T) {
	mfs := model.MediaFiles{
		{ID: "1", Title: "Song 1"},
		{ID: "2", Title: "Song 2"},
		{ID: "3", Title: "Song 3"},
	}

	songs := apimodel.FromMediaFiles(mfs)

	if len(songs) != 3 {
		t.Fatalf("expected 3 songs, got %d", len(songs))
	}
	for i, song := range songs {
		if song.ID != mfs[i].ID {
			t.Errorf("song[%d].ID: got %q, want %q", i, song.ID, mfs[i].ID)
		}
		if song.Title != mfs[i].Title {
			t.Errorf("song[%d].Title: got %q, want %q", i, song.Title, mfs[i].Title)
		}
	}
}

func TestFromMediaFilesEmpty(t *testing.T) {
	songs := apimodel.FromMediaFiles(model.MediaFiles{})
	if len(songs) != 0 {
		t.Fatalf("expected 0 songs, got %d", len(songs))
	}
}
