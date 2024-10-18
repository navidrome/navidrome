package tests

import (
	"context"
	"errors"
	"net/url"

	"github.com/navidrome/navidrome/model"
)

var (
	ErrMockMetadata = errors.New("mock metadata error")
)

type MockExternalMetadata struct {
	album  *model.Album
	artist *model.Artist
	err    bool
	lyrics model.LyricList
	mf     model.MediaFiles
	url    *url.URL
}

func CreateMockExternalMetadata() *MockExternalMetadata {
	return &MockExternalMetadata{}
}

func (m *MockExternalMetadata) SetAlbum(album *model.Album) {
	m.album = album
}

func (m *MockExternalMetadata) SetArtist(artist *model.Artist) {
	m.artist = artist
}

func (m *MockExternalMetadata) SetError(err bool) {
	m.err = err
}

func (m *MockExternalMetadata) SetLyrics(lyrics model.LyricList) {
	m.lyrics = lyrics
}

func (m *MockExternalMetadata) SetMediaFiles(mf model.MediaFiles) {
	m.mf = mf
}

func (m *MockExternalMetadata) SetUrl(url *url.URL) {
	m.url = url
}

func (m *MockExternalMetadata) AlbumImage(ctx context.Context, id string) (*url.URL, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.url, nil
}

func (m *MockExternalMetadata) ArtistImage(ctx context.Context, id string) (*url.URL, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.url, nil
}

func (m *MockExternalMetadata) ExternalLyrics(ctx context.Context, id string) (model.LyricList, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.lyrics, nil
}

func (m *MockExternalMetadata) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.mf, nil
}

func (m *MockExternalMetadata) TopSongs(ctx context.Context, artist string, count int) (model.MediaFiles, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.mf, nil
}

func (m *MockExternalMetadata) UpdateAlbumInfo(ctx context.Context, id string) (*model.Album, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.album, nil
}

func (m *MockExternalMetadata) UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error) {
	if m.err {
		return nil, ErrMockMetadata
	}

	return m.artist, nil
}
