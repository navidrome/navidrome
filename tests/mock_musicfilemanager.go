package tests

import (
	"context"
	"fmt"
	"io"
)

type MockMusicFileManager struct {
	MediaFileRepo *MockMediaFileRepo
}

func NewMockMusicFileManager(mfRepo *MockMediaFileRepo) *MockMusicFileManager {
	return &MockMusicFileManager{
		MediaFileRepo: mfRepo,
	}
}

func (m *MockMusicFileManager) DeleteSong(ctx context.Context, songID string) error {
	if m.MediaFileRepo == nil {
		return nil
	}

	if _, exists := m.MediaFileRepo.Data[songID]; !exists {
		return fmt.Errorf("song not found")
	}

	delete(m.MediaFileRepo.Data, songID)
	return nil
}

func (m *MockMusicFileManager) UpdateArtwork(ctx context.Context, songID string, data io.Reader, mimeType string) error {
	return nil
}

func (m *MockMusicFileManager) UpdateTags(ctx context.Context, songID string, tags map[string]string) error {
	return nil
}
