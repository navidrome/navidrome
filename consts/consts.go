package consts

import "time"

const (
	LocalConfigFile     = "./navidrome.toml"
	InitialSetupFlagKey = "InitialSetup"

	JWTSecretKey       = "JWTSecret"
	JWTIssuer          = "Navidrome"
	JWTTokenExpiration = 30 * time.Minute

	InitialUserName = "admin"
	InitialName     = "Admin"

	UIAssetsLocalPath = "ui/build"
)
