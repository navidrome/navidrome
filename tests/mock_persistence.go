package tests

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type MockDataStore struct {
	MockedGenre       model.GenreRepository
	MockedAlbum       model.AlbumRepository
	MockedArtist      model.ArtistRepository
	MockedMediaFile   model.MediaFileRepository
	MockedUser        model.UserRepository
	MockedProperty    model.PropertyRepository
	MockedPlayer      model.PlayerRepository
	MockedTranscoding model.TranscodingRepository
}

func (db *MockDataStore) Album(context.Context) model.AlbumRepository {
	if db.MockedAlbum == nil {
		db.MockedAlbum = CreateMockAlbumRepo()
	}
	return db.MockedAlbum
}

func (db *MockDataStore) Artist(context.Context) model.ArtistRepository {
	if db.MockedArtist == nil {
		db.MockedArtist = CreateMockArtistRepo()
	}
	return db.MockedArtist
}

func (db *MockDataStore) MediaFile(context.Context) model.MediaFileRepository {
	if db.MockedMediaFile == nil {
		db.MockedMediaFile = CreateMockMediaFileRepo()
	}
	return db.MockedMediaFile
}

func (db *MockDataStore) MediaFolder(context.Context) model.MediaFolderRepository {
	return struct{ model.MediaFolderRepository }{}
}

func (db *MockDataStore) Genre(context.Context) model.GenreRepository {
	if db.MockedGenre != nil {
		return db.MockedGenre
	}
	return struct{ model.GenreRepository }{}
}

func (db *MockDataStore) Playlist(context.Context) model.PlaylistRepository {
	return struct{ model.PlaylistRepository }{}
}

func (db *MockDataStore) PlayQueue(context.Context) model.PlayQueueRepository {
	return struct{ model.PlayQueueRepository }{}
}

func (db *MockDataStore) Property(context.Context) model.PropertyRepository {
	if db.MockedProperty == nil {
		db.MockedProperty = &mockedPropertyRepo{}
	}
	return db.MockedProperty
}

func (db *MockDataStore) User(context.Context) model.UserRepository {
	if db.MockedUser == nil {
		db.MockedUser = &mockedUserRepo{}
	}
	return db.MockedUser
}

func (db *MockDataStore) Transcoding(context.Context) model.TranscodingRepository {
	if db.MockedTranscoding != nil {
		return db.MockedTranscoding
	}
	return struct{ model.TranscodingRepository }{}
}

func (db *MockDataStore) Player(context.Context) model.PlayerRepository {
	if db.MockedPlayer != nil {
		return db.MockedPlayer
	}
	return struct{ model.PlayerRepository }{}
}

func (db *MockDataStore) WithTx(block func(db model.DataStore) error) error {
	return block(db)
}

func (db *MockDataStore) Resource(ctx context.Context, m interface{}) model.ResourceRepository {
	return struct{ model.ResourceRepository }{}
}

func (db *MockDataStore) GC(ctx context.Context, rootFolder string) error {
	return nil
}
