package responses

import (
	"encoding/xml"
)

const (
	ERROR_GENERIC = iota * 10
	ERROR_MISSING_PARAMETER
	ERROR_CLIENT_TOO_OLD
	ERROR_SERVER_TOO_OLD
	ERROR_AUTHENTICATION_FAIL
	ERROR_AUTHORIZATION_FAIL
	ERROR_TRIAL_EXPIRED
	ERROR_DATA_NOT_FOUND
)

var (
	errors map[int]string
)

func init() {
	errors = make(map[int]string)
	errors[ERROR_GENERIC] = "A generic error"
	errors[ERROR_MISSING_PARAMETER] = "Required parameter is missing"
	errors[ERROR_CLIENT_TOO_OLD] = "Incompatible Subsonic REST protocol version. Client must upgrade"
	errors[ERROR_SERVER_TOO_OLD] = "Incompatible Subsonic REST protocol version. Server must upgrade"
	errors[ERROR_AUTHENTICATION_FAIL] = "Wrong username or password"
	errors[ERROR_AUTHORIZATION_FAIL] = "User is not authorized for the given operation"
	errors[ERROR_TRIAL_EXPIRED] = "The trial period for the Subsonic server is over. Please upgrade to Subsonic Premium. Visit subsonic.org for details"
	errors[ERROR_DATA_NOT_FOUND] = "The requested data was not found"
}

type error struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}

func NewError(errorCode int) []byte {
	response := NewEmpty()
	response.Status = "fail"
	if errors[errorCode] == "" {
		errorCode = ERROR_GENERIC
	}
	xmlBody, _ := xml.Marshal(&error{Code: errorCode, Message: errors[errorCode]})
	response.Body = xmlBody
	xmlResponse, _ := xml.Marshal(response)
	return []byte(xml.Header + string(xmlResponse))
}
