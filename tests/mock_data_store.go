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
	MockedScrobble       model.ScrobbleRepository
	MockedRadio          model.RadioRepository
	MockedPlugin         model.PluginRepository
	scrobbleBufferMu     sync.Mutex
	repoMu               sync.Mutex

	// GC tracking
	GCCalled bool
	GCError  error
}

func (db *MockDataStore) Library(ctx context.Context) model.LibraryRepository {
	if db.MockedLibrary != nil {
		return db.MockedLibrary
	}
	if db.RealDS != nil {
		return db.RealDS.Library(ctx)
	}
	db.MockedLibrary = &MockLibraryRepo{}
	return db.MockedLibrary
}

func (db *MockDataStore) Folder(ctx context.Context) model.FolderRepository {
	if db.MockedFolder != nil {
		return db.MockedFolder
	}
	if db.RealDS != nil {
		return db.RealDS.Folder(ctx)
	}
	db.MockedFolder = struct{ model.FolderRepository }{}
	return db.MockedFolder
}

func (db *MockDataStore) Tag(ctx context.Context) model.TagRepository {
	if db.MockedTag != nil {
		return db.MockedTag
	}
	if db.RealDS != nil {
		return db.RealDS.Tag(ctx)
	}
	db.MockedTag = struct{ model.TagRepository }{}
	return db.MockedTag
}

func (db *MockDataStore) Album(ctx context.Context) model.AlbumRepository {
	if db.MockedAlbum != nil {
		return db.MockedAlbum
	}
	if db.RealDS != nil {
		return db.RealDS.Album(ctx)
	}
	db.MockedAlbum = CreateMockAlbumRepo()
	return db.MockedAlbum
}

func (db *MockDataStore) Artist(ctx context.Context) model.ArtistRepository {
	if db.MockedArtist != nil {
		return db.MockedArtist
	}
	if db.RealDS != nil {
		return db.RealDS.Artist(ctx)
	}
	db.MockedArtist = CreateMockArtistRepo()
	return db.MockedArtist
}

func (db *MockDataStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	if db.RealDS != nil && db.MockedMediaFile == nil {
		return db.RealDS.MediaFile(ctx)
	}
	db.repoMu.Lock()
	defer db.repoMu.Unlock()
	if db.MockedMediaFile == nil {
		db.MockedMediaFile = CreateMockMediaFileRepo()
	}
	return db.MockedMediaFile
}

func (db *MockDataStore) Genre(ctx context.Context) model.GenreRepository {
	if db.MockedGenre != nil {
		return db.MockedGenre
	}
	if db.RealDS != nil {
		return db.RealDS.Genre(ctx)
	}
	db.MockedGenre = &MockedGenreRepo{}
	return db.MockedGenre
}

func (db *MockDataStore) Playlist(ctx context.Context) model.PlaylistRepository {
	if db.MockedPlaylist != nil {
		return db.MockedPlaylist
	}
	if db.RealDS != nil {
		return db.RealDS.Playlist(ctx)
	}
	db.MockedPlaylist = CreateMockPlaylistRepo()
	return db.MockedPlaylist
}

func (db *MockDataStore) PlayQueue(ctx context.Context) model.PlayQueueRepository {
	if db.MockedPlayQueue != nil {
		return db.MockedPlayQueue
	}
	if db.RealDS != nil {
		return db.RealDS.PlayQueue(ctx)
	}
	db.MockedPlayQueue = &MockPlayQueueRepo{}
	return db.MockedPlayQueue
}

func (db *MockDataStore) UserProps(ctx context.Context) model.UserPropsRepository {
	if db.MockedUserProps != nil {
		return db.MockedUserProps
	}
	if db.RealDS != nil {
		return db.RealDS.UserProps(ctx)
	}
	db.MockedUserProps = &MockedUserPropsRepo{}
	return db.MockedUserProps
}

func (db *MockDataStore) Property(ctx context.Context) model.PropertyRepository {
	if db.MockedProperty != nil {
		return db.MockedProperty
	}
	if db.RealDS != nil {
		return db.RealDS.Property(ctx)
	}
	db.MockedProperty = &MockedPropertyRepo{}
	return db.MockedProperty
}

func (db *MockDataStore) Share(ctx context.Context) model.ShareRepository {
	if db.MockedShare != nil {
		return db.MockedShare
	}
	if db.RealDS != nil {
		return db.RealDS.Share(ctx)
	}
	db.MockedShare = &MockShareRepo{}
	return db.MockedShare
}

func (db *MockDataStore) User(ctx context.Context) model.UserRepository {
	if db.MockedUser != nil {
		return db.MockedUser
	}
	if db.RealDS != nil {
		return db.RealDS.User(ctx)
	}
	db.MockedUser = CreateMockUserRepo()
	return db.MockedUser
}

func (db *MockDataStore) Transcoding(ctx context.Context) model.TranscodingRepository {
	if db.MockedTranscoding != nil {
		return db.MockedTranscoding
	}
	if db.RealDS != nil {
		return db.RealDS.Transcoding(ctx)
	}
	db.MockedTranscoding = struct{ model.TranscodingRepository }{}
	return db.MockedTranscoding
}

func (db *MockDataStore) Player(ctx context.Context) model.PlayerRepository {
	if db.MockedPlayer != nil {
		return db.MockedPlayer
	}
	if db.RealDS != nil {
		return db.RealDS.Player(ctx)
	}
	db.MockedPlayer = struct{ model.PlayerRepository }{}
	return db.MockedPlayer
}

func (db *MockDataStore) ScrobbleBuffer(ctx context.Context) model.ScrobbleBufferRepository {
	if db.RealDS != nil && db.MockedScrobbleBuffer == nil {
		return db.RealDS.ScrobbleBuffer(ctx)
	}
	db.scrobbleBufferMu.Lock()
	defer db.scrobbleBufferMu.Unlock()
	if db.MockedScrobbleBuffer == nil {
		db.MockedScrobbleBuffer = &MockedScrobbleBufferRepo{}
	}
	return db.MockedScrobbleBuffer
}

func (db *MockDataStore) Scrobble(ctx context.Context) model.ScrobbleRepository {
	if db.MockedScrobble != nil {
		return db.MockedScrobble
	}
	if db.RealDS != nil {
		return db.RealDS.Scrobble(ctx)
	}
	db.MockedScrobble = &MockScrobbleRepo{ctx: ctx}
	return db.MockedScrobble
}

func (db *MockDataStore) Radio(ctx context.Context) model.RadioRepository {
	if db.MockedRadio != nil {
		return db.MockedRadio
	}
	if db.RealDS != nil {
		return db.RealDS.Radio(ctx)
	}
	db.MockedRadio = CreateMockedRadioRepo()
	return db.MockedRadio
}

func (db *MockDataStore) Plugin(ctx context.Context) model.PluginRepository {
	if db.MockedPlugin != nil {
		return db.MockedPlugin
	}
	if db.RealDS != nil {
		return db.RealDS.Plugin(ctx)
	}
	db.MockedPlugin = CreateMockPluginRepo()
	return db.MockedPlugin
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
	case model.Plugin, *model.Plugin:
		return db.Plugin(ctx).(model.ResourceRepository)
	default:
		return struct{ model.ResourceRepository }{}
	}
}

func (db *MockDataStore) GC(context.Context, ...int) error {
	db.GCCalled = true
	if db.GCError != nil {
		return db.GCError
	}
	return nil
}
