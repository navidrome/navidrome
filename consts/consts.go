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

	UIAuthorizationHeader = "X-ND-Authorization"
	JWTSecretKey          = "JWTSecret"
	JWTIssuer             = "ND"
	DefaultSessionTimeout = 24 * time.Hour

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"

	URLPathUI                   = "/app"
	URLPathSubsonicAPI          = "/rest"
	DefaultUILoginBackgroundURL = "https://source.unsplash.com/random/1600x900?music"

	RequestThrottleBacklogLimit   = 100
	RequestThrottleBacklogTimeout = time.Minute

	ArtistInfoTimeToLive = 1 * time.Hour

	I18nFolder   = "i18n"
	SkipScanFile = ".ndignore"

	PlaceholderAlbumArt = "navidrome-600x600.png"
	PlaceholderAvatar   = "logo-192x192.png"

	DefaultCachedHttpClientTTL = 10 * time.Second
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
