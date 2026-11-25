package persistencetest

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type overlay struct {
	model.DataStore
	wrap func(model.DataStore) model.DataStore
}

// Overlay returns wrap(ds), and also wraps every transaction ds opens, so a stub that overrides
// one repository still applies inside WithTx. wrap's result must embed the DataStore it receives.
func Overlay(ds model.DataStore, wrap func(model.DataStore) model.DataStore) model.DataStore {
	return wrap(&overlay{DataStore: ds, wrap: wrap})
}

func (o *overlay) WithTx(block func(tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTx(func(tx model.DataStore) error {
		return block(Overlay(tx, o.wrap))
	}, scope...)
}

func (o *overlay) WithTxImmediate(block func(tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTxImmediate(func(tx model.DataStore) error {
		return block(Overlay(tx, o.wrap))
	}, scope...)
}

func (o *overlay) WithTxRetry(ctx context.Context, block func(ctx context.Context, tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
		return block(ctx, Overlay(tx, o.wrap))
	}, scope...)
}

// Repos holds optional wrappers for the repositories of a DataStore, one per accessor. Each
// receives the repository the accessor would return, the transaction's one inside a transaction.
type Repos struct {
	Library        func(model.LibraryRepository) model.LibraryRepository
	Folder         func(model.FolderRepository) model.FolderRepository
	Album          func(model.AlbumRepository) model.AlbumRepository
	Artist         func(model.ArtistRepository) model.ArtistRepository
	MediaFile      func(model.MediaFileRepository) model.MediaFileRepository
	Genre          func(model.GenreRepository) model.GenreRepository
	Tag            func(model.TagRepository) model.TagRepository
	Playlist       func(model.PlaylistRepository) model.PlaylistRepository
	PlayQueue      func(model.PlayQueueRepository) model.PlayQueueRepository
	Transcoding    func(model.TranscodingRepository) model.TranscodingRepository
	Player         func(model.PlayerRepository) model.PlayerRepository
	Radio          func(model.RadioRepository) model.RadioRepository
	Share          func(model.ShareRepository) model.ShareRepository
	Property       func(model.PropertyRepository) model.PropertyRepository
	User           func(model.UserRepository) model.UserRepository
	UserProps      func(model.UserPropsRepository) model.UserPropsRepository
	ScrobbleBuffer func(model.ScrobbleBufferRepository) model.ScrobbleBufferRepository
	Scrobble       func(model.ScrobbleRepository) model.ScrobbleRepository
	Plugin         func(model.PluginRepository) model.PluginRepository
	Artwork        func(model.ArtworkRepository) model.ArtworkRepository
	ArtworkQueue   func(model.ArtworkQueueRepository) model.ArtworkQueueRepository
	Grant          func(model.GrantRepository) model.GrantRepository
}

// Stub returns ds with the non-nil Repos wrappers applied, also inside every transaction it opens.
// A wrapper runs on every accessor call, so state shared across calls must live outside the repository it returns.
func Stub(ds model.DataStore, repos Repos) model.DataStore {
	return Overlay(ds, func(ds model.DataStore) model.DataStore { return &stubbed{DataStore: ds, repos: repos} })
}

type stubbed struct {
	model.DataStore
	repos Repos
}

func wrapRepo[R any](wrap func(R) R, repo R) R {
	if wrap == nil {
		return repo
	}
	return wrap(repo)
}

func (s *stubbed) Library() model.LibraryRepository {
	return wrapRepo(s.repos.Library, s.DataStore.Library())
}

func (s *stubbed) Folder() model.FolderRepository {
	return wrapRepo(s.repos.Folder, s.DataStore.Folder())
}

func (s *stubbed) Album() model.AlbumRepository {
	return wrapRepo(s.repos.Album, s.DataStore.Album())
}

func (s *stubbed) Artist() model.ArtistRepository {
	return wrapRepo(s.repos.Artist, s.DataStore.Artist())
}

func (s *stubbed) MediaFile() model.MediaFileRepository {
	return wrapRepo(s.repos.MediaFile, s.DataStore.MediaFile())
}

func (s *stubbed) Genre() model.GenreRepository {
	return wrapRepo(s.repos.Genre, s.DataStore.Genre())
}

func (s *stubbed) Tag() model.TagRepository {
	return wrapRepo(s.repos.Tag, s.DataStore.Tag())
}

func (s *stubbed) Playlist() model.PlaylistRepository {
	return wrapRepo(s.repos.Playlist, s.DataStore.Playlist())
}

func (s *stubbed) PlayQueue() model.PlayQueueRepository {
	return wrapRepo(s.repos.PlayQueue, s.DataStore.PlayQueue())
}

func (s *stubbed) Transcoding() model.TranscodingRepository {
	return wrapRepo(s.repos.Transcoding, s.DataStore.Transcoding())
}

func (s *stubbed) Player() model.PlayerRepository {
	return wrapRepo(s.repos.Player, s.DataStore.Player())
}

func (s *stubbed) Radio() model.RadioRepository {
	return wrapRepo(s.repos.Radio, s.DataStore.Radio())
}

func (s *stubbed) Share() model.ShareRepository {
	return wrapRepo(s.repos.Share, s.DataStore.Share())
}

func (s *stubbed) Property() model.PropertyRepository {
	return wrapRepo(s.repos.Property, s.DataStore.Property())
}

func (s *stubbed) User() model.UserRepository {
	return wrapRepo(s.repos.User, s.DataStore.User())
}

func (s *stubbed) UserProps() model.UserPropsRepository {
	return wrapRepo(s.repos.UserProps, s.DataStore.UserProps())
}

func (s *stubbed) ScrobbleBuffer() model.ScrobbleBufferRepository {
	return wrapRepo(s.repos.ScrobbleBuffer, s.DataStore.ScrobbleBuffer())
}

func (s *stubbed) Scrobble() model.ScrobbleRepository {
	return wrapRepo(s.repos.Scrobble, s.DataStore.Scrobble())
}

func (s *stubbed) Plugin() model.PluginRepository {
	return wrapRepo(s.repos.Plugin, s.DataStore.Plugin())
}

func (s *stubbed) Artwork() model.ArtworkRepository {
	return wrapRepo(s.repos.Artwork, s.DataStore.Artwork())
}

func (s *stubbed) ArtworkQueue() model.ArtworkQueueRepository {
	return wrapRepo(s.repos.ArtworkQueue, s.DataStore.ArtworkQueue())
}

func (s *stubbed) Grant() model.GrantRepository {
	return wrapRepo(s.repos.Grant, s.DataStore.Grant())
}
