package consts

import "time"

const (
	AppName = "navidrome"

	LocalConfigFile     = "./navidrome.toml"
	DefaultDbPath       = "navidrome.db?cache=shared&_busy_timeout=15000&_journal_mode=WAL"
	InitialSetupFlagKey = "InitialSetup"

	JWTSecretKey       = "JWTSecret"
	JWTIssuer          = "ND"
	JWTTokenExpiration = 30 * time.Minute

	UIAssetsLocalPath = "ui/build"

	CacheDir = "cache"

	DevInitialUserName = "admin"
	DevInitialName     = "Dev Admin"
)

var (
	DefaultTranscodings = []map[string]interface{}{
		{
			"name":           "mp3 audio",
			"targetFormat":   "mp3",
			"defaultBitRate": 192,
			"command":        "ffmpeg -i %s -ab %bk -v 0 -f mp3 -",
		},
		{
			"name":           "opus audio",
			"targetFormat":   "oga",
			"defaultBitRate": 128,
			"command":        "ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		},
	}
)
