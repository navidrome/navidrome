package consts

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppName = "navidrome"

	DefaultDbPath       = "navidrome.db?cache=shared&_busy_timeout=15000&_journal_mode=WAL&_foreign_keys=on"
	InitialSetupFlagKey = "InitialSetup"

	UIAuthorizationHeader  = "X-ND-Authorization"
	UIClientUniqueIDHeader = "X-ND-Client-Unique-Id"
	JWTSecretKey           = "JWTSecret"
	JWTIssuer              = "ND"
	DefaultSessionTimeout  = 24 * time.Hour
	CookieExpiry           = 365 * 24 * 3600 // One year

	// DefaultEncryptionKey This is the encryption key used if none is specified in the `PasswordEncryptionKey` option
	// Never ever change this! Or it will break all Navidrome installations that don't set the config option
	DefaultEncryptionKey  = "just for obfuscation"
	PasswordsEncryptedKey = "PasswordsEncryptedKey"

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"

	URLPathUI           = "/app"
	URLPathNativeAPI    = "/api"
	URLPathSubsonicAPI  = "/rest"
	URLPathPublic       = "/p"
	URLPathPublicImages = URLPathPublic + "/img"

	// DefaultUILoginBackgroundURL uses Navidrome curated background images collection,
	// available at https://unsplash.com/collections/20072696/navidrome
	DefaultUILoginBackgroundURL = "/backgrounds"

	// DefaultUILoginBackgroundOffline Background image used in case external integrations are disabled
	DefaultUILoginBackgroundOffline    = "iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAIAAAAiOjnJAAAABGdBTUEAALGPC/xhBQAAAiJJREFUeF7t0IEAAAAAw6D5Ux/khVBhwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDBgwIABAwYMGDDwMDDVlwABBWcSrQAAAABJRU5ErkJggg=="
	DefaultUILoginBackgroundURLOffline = "data:image/png;base64," + DefaultUILoginBackgroundOffline

	RequestThrottleBacklogLimit   = 100
	RequestThrottleBacklogTimeout = time.Minute

	ServerReadHeaderTimeout = 3 * time.Second

	ArtistInfoTimeToLive = time.Second // TODO Revert
	//ArtistInfoTimeToLive = 24 * time.Hour

	I18nFolder   = "i18n"
	SkipScanFile = ".ndignore"

	PlaceholderArtistArt = "artist-placeholder.webp"
	PlaceholderAlbumArt  = "placeholder.png"
	PlaceholderAvatar    = "logo-192x192.png"

	DefaultUIVolume = 100

	DefaultHttpClientTimeOut = 10 * time.Second

	DefaultScannerExtractor = "taglib"

	Zwsp = string('\u200b')
)

// Cache options
const (
	TranscodingCacheDir             = "cache/transcoding"
	DefaultTranscodingCacheMaxItems = 0 // Unlimited

	ImageCacheDir             = "cache/images"
	DefaultImageCacheMaxItems = 0 // Unlimited

	DefaultCacheSize            = 100 * 1024 * 1024 // 100MB
	DefaultCacheCleanUpInterval = 10 * time.Minute
)

// Shared secrets (only add here "secrets" that can be public)
const (
	LastFMAPIKey    = "9b94a5515ea66b2da3ec03c12300327e" // nolint:gosec
	LastFMAPISecret = "74cb6557cec7171d921af5d7d887c587" // nolint:gosec
)

var (
	DefaultDownsamplingFormat = "opus"
	DefaultTranscodings       = []map[string]interface{}{
		{
			"name":           "mp3 audio",
			"targetFormat":   "mp3",
			"defaultBitRate": 192,
			"command":        "ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -f mp3 -",
		},
		{
			"name":           "opus audio",
			"targetFormat":   "opus",
			"defaultBitRate": 128,
			"command":        "ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		},
		{
			"name":           "aac audio",
			"targetFormat":   "aac",
			"defaultBitRate": 256,
			"command":        "ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -c:a aac -f adts -",
		},
	}

	DefaultPlaylistsPath = strings.Join([]string{".", "**/**"}, string(filepath.ListSeparator))
)

var (
	VariousArtists      = "Various Artists"
	VariousArtistsID    = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(VariousArtists))))
	UnknownArtist       = "[Unknown Artist]"
	VariousArtistsMbzId = "89ad4ac3-39f7-470e-963a-56509c546377"

	ServerStart = time.Now()
)
