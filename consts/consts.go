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

	JWTSecretKey          = "JWTSecret"
	JWTIssuer             = "ND"
	DefaultSessionTimeout = 30 * time.Minute

	UIAssetsLocalPath = "ui/build"

	TranscodingCacheDir                    = "cache/transcoding"
	DefaultTranscodingCacheSize            = 100 * 1024 * 1024 // 100MB
	DefaultTranscodingCacheMaxItems        = 0                 // Unlimited
	DefaultTranscodingCacheCleanUpInterval = 10 * time.Minute

	ImageCacheDir                    = "cache/images"
	DefaultImageCacheSize            = 100 * 1024 * 1024 // 100MB
	DefaultImageCacheMaxItems        = 0                 // Unlimited
	DefaultImageCacheCleanUpInterval = 10 * time.Minute

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"

	URLPathUI          = "/app"
	URLPathSubsonicAPI = "/rest"
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
	UnknownArtistID  = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(UnknownArtist))))
)
