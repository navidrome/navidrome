package consts

import "time"

const (
	InitialSetupFlagKey = "InitialSetupKey"

	JWTSecretKey       = "JWTSecretKey"
	JWTIssuer          = "CloudSonic"
	JWTTokenExpiration = 30 * time.Minute

	InitialUserName = "admin"
	InitialName     = "Admin"
)
