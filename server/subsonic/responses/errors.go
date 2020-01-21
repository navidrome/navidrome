package responses

const (
	ErrorGeneric            = 0
	ErrorMissingParameter   = 10
	ErrorClientTooOld       = 20
	ErrorServerTooOld       = 30
	ErrorAuthenticationFail = 40
	ErrorAuthorizationFail  = 50
	ErrorTrialExpired       = 60
	ErrorDataNotFound       = 70
)

var errors = map[int]string{
	ErrorGeneric:            "A generic error",
	ErrorMissingParameter:   "Required parameter is missing",
	ErrorClientTooOld:       "Incompatible Subsonic REST protocol version. Client must upgrade",
	ErrorServerTooOld:       "Incompatible Subsonic REST protocol version. Server must upgrade",
	ErrorAuthenticationFail: "Wrong username or password",
	ErrorAuthorizationFail:  "User is not authorized for the given operation",
	ErrorTrialExpired:       "The trial period for the Subsonic server is over. Please upgrade to Subsonic Premium. Visit subsonic.org for details",
	ErrorDataNotFound:       "The requested data was not found",
}

func ErrorMsg(code int) string {
	if v, found := errors[code]; found {
		return v
	}
	return errors[ErrorGeneric]
}
