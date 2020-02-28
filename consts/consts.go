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
