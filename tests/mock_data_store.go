package tests

import (
	"context"
	"sync"

	"github.com/navidrome/navidrome/model"
)

type MockDataStore struct {
	RealDS               model.DataStore
	MockedLibrary        model.LibraryRepository
	MockedFolder         model.FolderRepository
	MockedGenre          model.GenreRepository
	MockedAlbum          model.AlbumRepository
	MockedArtist         model.ArtistRepository
	MockedMediaFile      model.MediaFileRepository
	MockedTag            model.TagRepository
	MockedUser           model.UserRepository
	MockedProperty       model.PropertyRepository
	MockedPlayer         model.PlayerRepository
	MockedPlaylist       model.PlaylistRepository
	MockedPlayQueue      model.PlayQueueRepository
	MockedShare          model.ShareRepository
	MockedTranscoding    model.TranscodingRepository
	MockedUserProps      model.UserPropsRepository
	MockedScrobbleBuffer model.ScrobbleBufferRepository
	MockedRadio          model.RadioRepository
	scrobbleBufferMu     sync.Mutex
	repoMu               sync.Mutex
}

func (db *MockDataStore) Library(ctx context.Context) model.LibraryRepository {
	if db.MockedLibrary == nil {
		if db.RealDS != nil {
			db.MockedLibrary = db.RealDS.Library(ctx)
		} else {
			db.MockedLibrary = &MockLibraryRepo{}
		}
	}
	return db.MockedLibrary
}

func (db *MockDataStore) Folder(ctx context.Context) model.FolderRepository {
	if db.MockedFolder == nil {
		if db.RealDS != nil {
			db.MockedFolder = db.RealDS.Folder(ctx)
		} else {
			db.MockedFolder = struct{ model.FolderRepository }{}
		}
	}
	return db.MockedFolder
}

func (db *MockDataStore) Tag(ctx context.Context) model.TagRepository {
	if db.MockedTag == nil {
		if db.RealDS != nil {
			db.MockedTag = db.RealDS.Tag(ctx)
		} else {
			db.MockedTag = struct{ model.TagRepository }{}
		}
	}
	return db.MockedTag
}

func (db *MockDataStore) Album(ctx context.Context) model.AlbumRepository {
	if db.MockedAlbum == nil {
		if db.RealDS != nil {
			db.MockedAlbum = db.RealDS.Album(ctx)
		} else {
			db.MockedAlbum = CreateMockAlbumRepo()
		}
	}
	return db.MockedAlbum
}

func (db *MockDataStore) Artist(ctx context.Context) model.ArtistRepository {
	if db.MockedArtist == nil {
		if db.RealDS != nil {
			db.MockedArtist = db.RealDS.Artist(ctx)
		} else {
			db.MockedArtist = CreateMockArtistRepo()
		}
	}
	return db.MockedArtist
}

func (db *MockDataStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	db.repoMu.Lock()
	defer db.repoMu.Unlock()
	if db.MockedMediaFile == nil {
		if db.RealDS != nil {
			db.MockedMediaFile = db.RealDS.MediaFile(ctx)
		} else {
			db.MockedMediaFile = CreateMockMediaFileRepo()
		}
	}
	return db.MockedMediaFile
}

func (db *MockDataStore) Genre(ctx context.Context) model.GenreRepository {
	if db.MockedGenre == nil {
		if db.RealDS != nil {
			db.MockedGenre = db.RealDS.Genre(ctx)
		} else {
			db.MockedGenre = &MockedGenreRepo{}
		}
	}
	return db.MockedGenre
}

func (db *MockDataStore) Playlist(ctx context.Context) model.PlaylistRepository {
	if db.MockedPlaylist == nil {
		if db.RealDS != nil {
			db.MockedPlaylist = db.RealDS.Playlist(ctx)
		} else {
			db.MockedPlaylist = &MockPlaylistRepo{}
		}
	}
	return db.MockedPlaylist
}

func (db *MockDataStore) PlayQueue(ctx context.Context) model.PlayQueueRepository {
	if db.MockedPlayQueue == nil {
		if db.RealDS != nil {
			db.MockedPlayQueue = db.RealDS.PlayQueue(ctx)
		} else {
			db.MockedPlayQueue = &MockPlayQueueRepo{}
		}
	}
	return db.MockedPlayQueue
}

func (db *MockDataStore) UserProps(ctx context.Context) model.UserPropsRepository {
	if db.MockedUserProps == nil {
		if db.RealDS != nil {
			db.MockedUserProps = db.RealDS.UserProps(ctx)
		} else {
			db.MockedUserProps = &MockedUserPropsRepo{}
		}
	}
	return db.MockedUserProps
}

func (db *MockDataStore) Property(ctx context.Context) model.PropertyRepository {
	if db.MockedProperty == nil {
		if db.RealDS != nil {
			db.MockedProperty = db.RealDS.Property(ctx)
		} else {
			db.MockedProperty = &MockedPropertyRepo{}
		}
	}
	return db.MockedProperty
}

func (db *MockDataStore) Share(ctx context.Context) model.ShareRepository {
	if db.MockedShare == nil {
		if db.RealDS != nil {
			db.MockedShare = db.RealDS.Share(ctx)
		} else {
			db.MockedShare = &MockShareRepo{}
		}
	}
	return db.MockedShare
}

func (db *MockDataStore) User(ctx context.Context) model.UserRepository {
	if db.MockedUser == nil {
		if db.RealDS != nil {
			db.MockedUser = db.RealDS.User(ctx)
		} else {
			db.MockedUser = CreateMockUserRepo()
		}
	}
	return db.MockedUser
}

func (db *MockDataStore) Transcoding(ctx context.Context) model.TranscodingRepository {
	if db.MockedTranscoding == nil {
		if db.RealDS != nil {
			db.MockedTranscoding = db.RealDS.Transcoding(ctx)
		} else {
			db.MockedTranscoding = struct{ model.TranscodingRepository }{}
		}
	}
	return db.MockedTranscoding
}

func (db *MockDataStore) Player(ctx context.Context) model.PlayerRepository {
	if db.MockedPlayer == nil {
		if db.RealDS != nil {
			db.MockedPlayer = db.RealDS.Player(ctx)
		} else {
			db.MockedPlayer = struct{ model.PlayerRepository }{}
		}
	}
	return db.MockedPlayer
}

func (db *MockDataStore) ScrobbleBuffer(ctx context.Context) model.ScrobbleBufferRepository {
	db.scrobbleBufferMu.Lock()
	defer db.scrobbleBufferMu.Unlock()
	if db.MockedScrobbleBuffer == nil {
		if db.RealDS != nil {
			db.MockedScrobbleBuffer = db.RealDS.ScrobbleBuffer(ctx)
		} else {
			db.MockedScrobbleBuffer = CreateMockedScrobbleBufferRepo()
		}
	}
	return db.MockedScrobbleBuffer
}

func (db *MockDataStore) Radio(ctx context.Context) model.RadioRepository {
	if db.MockedRadio == nil {
		if db.RealDS != nil {
			db.MockedRadio = db.RealDS.Radio(ctx)
		} else {
			db.MockedRadio = CreateMockedRadioRepo()
		}
	}
	return db.MockedRadio
}

func (db *MockDataStore) WithTx(block func(tx model.DataStore) error, label ...string) error {
	return block(db)
}

func (db *MockDataStore) WithTxImmediate(block func(tx model.DataStore) error, label ...string) error {
	return block(db)
}

func (db *MockDataStore) Resource(ctx context.Context, m any) model.ResourceRepository {
	switch m.(type) {
	case model.MediaFile, *model.MediaFile:
		return db.MediaFile(ctx).(model.ResourceRepository)
	case model.Album, *model.Album:
		return db.Album(ctx).(model.ResourceRepository)
	case model.Artist, *model.Artist:
		return db.Artist(ctx).(model.ResourceRepository)
	case model.User, *model.User:
		return db.User(ctx).(model.ResourceRepository)
	case model.Playlist, *model.Playlist:
		return db.Playlist(ctx).(model.ResourceRepository)
	case model.Radio, *model.Radio:
		return db.Radio(ctx).(model.ResourceRepository)
	case model.Share, *model.Share:
		return db.Share(ctx).(model.ResourceRepository)
	case model.Genre, *model.Genre:
		return db.Genre(ctx).(model.ResourceRepository)
	case model.Tag, *model.Tag:
		return db.Tag(ctx).(model.ResourceRepository)
	case model.Transcoding, *model.Transcoding:
		return db.Transcoding(ctx).(model.ResourceRepository)
	case model.Player, *model.Player:
		return db.Player(ctx).(model.ResourceRepository)
	default:
		return struct{ model.ResourceRepository }{}
	}
}

func (db *MockDataStore) GC(context.Context) error {
	return nil
}
