package consts

import "time"

const (
	AppName = "navidrome"

	LocalConfigFile     = "./navidrome.toml"
	InitialSetupFlagKey = "InitialSetup"

	JWTSecretKey       = "JWTSecret"
	JWTIssuer          = "ND"
	JWTTokenExpiration = 30 * time.Minute

	UIAssetsLocalPath = "ui/build"
)
