package responses

type valid struct {
	Valid bool `xml:"valid,attr"`
}

type license struct {
	Subsonic
	Body valid `xml:"license"`
}

func NewGetLicense(valid bool) *license {
	response := new(license)
	response.Subsonic = NewSubsonic()
	response.Body.Valid = valid
	return response
}
