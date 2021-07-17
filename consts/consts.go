package consts

import (
	"crypto/md5"
	"fmt"
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

	URLPathUI          = "/app"
	URLPathNativeAPI   = "/api"
	URLPathSubsonicAPI = "/rest"

	// Login backgrounds from https://unsplash.com/collections/20072696/navidrome
	DefaultUILoginBackgroundURL = "https://source.unsplash.com/collection/20072696/1600x900"

	RequestThrottleBacklogLimit   = 100
	RequestThrottleBacklogTimeout = time.Minute

	ArtistInfoTimeToLive = 3 * 24 * time.Hour

	I18nFolder   = "i18n"
	SkipScanFile = ".ndignore"

	PlaceholderAlbumArt = "navidrome-600x600.png"
	PlaceholderAvatar   = "logo-192x192.png"

	DefaultHttpClientTimeOut = 10 * time.Second
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
	LastFMAPIKey    = "9b94a5515ea66b2da3ec03c12300327e"
	LastFMAPISecret = "74cb6557cec7171d921af5d7d887c587" // nolint:gosec
)

var (
	DefaultTranscodings = []map[string]interface{}{
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
	}
)

var (
	VariousArtists   = "Various Artists"
	VariousArtistsID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(VariousArtists))))
	UnknownArtist    = "[Unknown Artist]"

	ServerStart = time.Now()
)
