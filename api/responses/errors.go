package responses

const (
	ErrorGeneric = iota * 10
	ErrorMissingParameter
	ErrorClientTooOld
	ErrorServerTooOld
	ErrorAuthenticationFail
	ErrorAuthorizationFail
	ErrorTrialExpired
	ErrorDataNotFound
)

var (
	errors map[int]string
)

func init() {
	errors = make(map[int]string)
	errors[ErrorGeneric] = "A generic error"
	errors[ErrorMissingParameter] = "Required parameter is missing"
	errors[ErrorClientTooOld] = "Incompatible Subsonic REST protocol version. Client must upgrade"
	errors[ErrorServerTooOld] = "Incompatible Subsonic REST protocol version. Server must upgrade"
	errors[ErrorAuthenticationFail] = "Wrong username or password"
	errors[ErrorAuthorizationFail] = "User is not authorized for the given operation"
	errors[ErrorTrialExpired] = "The trial period for the Subsonic server is over. Please upgrade to Subsonic Premium. Visit subsonic.org for details"
	errors[ErrorDataNotFound] = "The requested data was not found"
}

func ErrorMsg(code int) string {
	if v, found := errors[code]; found {
		return v
	}
	return errors[ErrorGeneric]
}
