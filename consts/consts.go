package consts

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"
)

const (
	AppName = "navidrome"

	LocalConfigFile     = "./navidrome.toml"
	DefaultDbPath       = "navidrome.db?cache=shared&_busy_timeout=15000&_journal_mode=WAL"
	InitialSetupFlagKey = "InitialSetup"

	UIAuthorizationHeader = "X-ND-Authorization"
	JWTSecretKey          = "JWTSecret"
	JWTIssuer             = "ND"
	DefaultSessionTimeout = 30 * time.Minute

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"

	URLPathUI          = "/app"
	URLPathSubsonicAPI = "/rest"

	RequestThrottleBacklogLimit   = 100
	RequestThrottleBacklogTimeout = time.Minute

	I18nFolder   = "i18n"
	SkipScanFile = ".ndignore"

	PlaceholderAlbumArt = "navidrome-600x600.png"
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
			"targetFormat":   "oga",
			"defaultBitRate": 128,
			"command":        "ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		},
	}
)

var (
	VariousArtists   = "Various Artists"
	VariousArtistsID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(VariousArtists))))
	UnknownArtist    = "[Unknown Artist]"
)
