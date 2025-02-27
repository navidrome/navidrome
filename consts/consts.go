package consts

import (
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model/id"
)

const (
	AppName = "navidrome"

	DefaultDbPath                 = "navidrome.db?cache=shared&_busy_timeout=15000&_journal_mode=WAL&_foreign_keys=on&synchronous=normal"
	InitialSetupFlagKey           = "InitialSetup"
	FullScanAfterMigrationFlagKey = "FullScanAfterMigration"

	UIAuthorizationHeader  = "X-ND-Authorization"
	UIClientUniqueIDHeader = "X-ND-Client-Unique-Id"
	JWTSecretKey           = "JWTSecret"
	JWTIssuer              = "ND"
	DefaultSessionTimeout  = 48 * time.Hour
	CookieExpiry           = 365 * 24 * 3600 // One year

	OptimizeDBSchedule = "@every 24h"

	// DefaultEncryptionKey This is the encryption key used if none is specified in the `PasswordEncryptionKey` option
	// Never ever change this! Or it will break all Navidrome installations that don't set the config option
	DefaultEncryptionKey  = "just for obfuscation"
	PasswordsEncryptedKey = "PasswordsEncryptedKey"
	PasswordAutogenPrefix = "__NAVIDROME_AUTOGEN__" //nolint:gosec

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"

	URLPathUI           = "/app"
	URLPathNativeAPI    = "/api"
	URLPathSubsonicAPI  = "/rest"
	URLPathPublic       = "/share"
	URLPathPublicImages = URLPathPublic + "/img"

	// DefaultUILoginBackgroundURL uses Navidrome curated background images collection,
	// available at https://unsplash.com/collections/20072696/navidrome
	DefaultUILoginBackgroundURL = "/backgrounds"

	// DefaultUILoginBackgroundOffline Background image used in case external integrations are disabled
	DefaultUILoginBackgroundOffline    = "iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAIAAAAiOjnJAAAABGdBTUEAALGPC/xhBQAAAiJJREFUeF7t0IEAAAAAw6D5Ux/khVBhwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDDwMDDVlwABBWcSrQAAAABJRU5ErkJggg=="
	DefaultUILoginBackgroundURLOffline = "data:image/png;base64," + DefaultUILoginBackgroundOffline
	DefaultMaxSidebarPlaylists         = 100

	RequestThrottleBacklogLimit   = 100
	RequestThrottleBacklogTimeout = time.Minute

	ServerReadHeaderTimeout = 3 * time.Second

	ArtistInfoTimeToLive      = 24 * time.Hour
	AlbumInfoTimeToLive       = 7 * 24 * time.Hour
	UpdateLastAccessFrequency = time.Minute
	UpdatePlayerFrequency     = time.Minute

	I18nFolder     = "i18n"
	ScanIgnoreFile = ".ndignore"

	PlaceholderArtistArt = "artist-placeholder.webp"
	PlaceholderAlbumArt  = "album-placeholder.webp"
	PlaceholderAvatar    = "logo-192x192.png"
	UICoverArtSize       = 300
	DefaultUIVolume      = 100

	DefaultHttpClientTimeOut = 10 * time.Second

	DefaultScannerExtractor = "taglib"
	DefaultWatcherWait      = 5 * time.Second
	Zwsp                    = string('\u200b')
)

// Prometheus options
const (
	PrometheusDefaultPath = "/metrics"
	PrometheusAuthUser    = "navidrome"
)

// Cache options
const (
	TranscodingCacheDir             = "transcoding"
	DefaultTranscodingCacheMaxItems = 0 // Unlimited

	ImageCacheDir             = "images"
	DefaultImageCacheMaxItems = 0 // Unlimited

	DefaultCacheSize            = 100 * 1024 * 1024 // 100MB
	DefaultCacheCleanUpInterval = 10 * time.Minute
)

const (
	AlbumPlayCountModeAbsolute   = "absolute"
	AlbumPlayCountModeNormalized = "normalized"
)

const (
	//DefaultAlbumPID = "album_legacy"
	DefaultAlbumPID = "musicbrainz_albumid|albumartistid,album,albumversion,releasedate"
	DefaultTrackPID = "musicbrainz_trackid|albumid,discnumber,tracknumber,title"
	PIDAlbumKey     = "PIDAlbum"
	PIDTrackKey     = "PIDTrack"
)

const (
	InsightsIDKey          = "InsightsID"
	InsightsEndpoint       = "https://insights.navidrome.org/collect"
	InsightsUpdateInterval = 24 * time.Hour
	InsightsInitialDelay   = 30 * time.Minute
)

var (
	DefaultDownsamplingFormat = "opus"
	DefaultTranscodings       = []struct {
		Name           string
		TargetFormat   string
		DefaultBitRate int
		Command        string
	}{
		{
			Name:           "mp3 audio",
			TargetFormat:   "mp3",
			DefaultBitRate: 192,
			Command:        "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -f mp3 -",
		},
		{
			Name:           "opus audio",
			TargetFormat:   "opus",
			DefaultBitRate: 128,
			Command:        "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		},
		{
			Name:           "aac audio",
			TargetFormat:   "aac",
			DefaultBitRate: 256,
			Command:        "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
		},
	}
)

var (
	VariousArtists = "Various Artists"
	// TODO This will be dynamic when using disambiguation
	VariousArtistsID = "63sqASlAfjbGMuLP4JhnZU"
	UnknownAlbum     = "[Unknown Album]"
	UnknownArtist    = "[Unknown Artist]"
	// TODO This will be dynamic when using disambiguation
	UnknownArtistID     = id.NewHash(strings.ToLower(UnknownArtist))
	VariousArtistsMbzId = "89ad4ac3-39f7-470e-963a-56509c546377"

	ServerStart = time.Now()
)

var InContainer = func() bool {
	// Check if the /.nddockerenv file exists
	if _, err := os.Stat("/.nddockerenv"); err == nil {
		return true
	}
	return false
}()
