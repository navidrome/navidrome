package consts

import "time"

const (
	LocalConfigFile     = "./navidrome.toml"
	InitialSetupFlagKey = "InitialSetup"

	JWTSecretKey       = "JWTSecret"
	JWTIssuer          = "ND"
	JWTTokenExpiration = 30 * time.Minute

	InitialUserName = "admin"

	UIAssetsLocalPath = "ui/build"
)
