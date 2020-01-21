package consts

import "time"

const (
	InitialSetupFlagKey = "InitialSetup"

	JWTSecretKey       = "JWTSecret"
	JWTIssuer          = "CloudSonic"
	JWTTokenExpiration = 30 * time.Minute

	InitialUserName = "admin"
	InitialName     = "Admin"
)
