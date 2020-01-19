package persistence

import "github.com/cloudsonic/sonic-server/model"

type MockDataStore struct {
	MockedGenre     model.GenreRepository
	MockedAlbum     model.AlbumRepository
	MockedArtist    model.ArtistRepository
	MockedMediaFile model.MediaFileRepository
}

func (db *MockDataStore) Album() model.AlbumRepository {
	if db.MockedAlbum == nil {
		db.MockedAlbum = CreateMockAlbumRepo()
	}
	return db.MockedAlbum
}

func (db *MockDataStore) Artist() model.ArtistRepository {
	if db.MockedArtist == nil {
		db.MockedArtist = CreateMockArtistRepo()
	}
	return db.MockedArtist
}

func (db *MockDataStore) MediaFile() model.MediaFileRepository {
	if db.MockedMediaFile == nil {
		db.MockedMediaFile = CreateMockMediaFileRepo()
	}
	return db.MockedMediaFile
}

func (db *MockDataStore) MediaFolder() model.MediaFolderRepository {
	return struct{ model.MediaFolderRepository }{}
}

func (db *MockDataStore) Genre() model.GenreRepository {
	if db.MockedGenre != nil {
		return db.MockedGenre
	}
	return struct{ model.GenreRepository }{}
}

func (db *MockDataStore) Playlist() model.PlaylistRepository {
	return struct{ model.PlaylistRepository }{}
}

func (db *MockDataStore) Property() model.PropertyRepository {
	return struct{ model.PropertyRepository }{}
}

func (db *MockDataStore) WithTx(block func(db model.DataStore) error) error {
	return block(db)
}
