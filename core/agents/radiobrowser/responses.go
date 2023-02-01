package radiobrowser

import "time"

type RadioStation struct {
	ID                    string     `json:"id"`
	ChangeId              string     `json:"changeuuid"`
	StationID             string     `json:"stationuuid"`
	ServerId              *string    `json:"serveruuid,omitempty"`
	Name                  string     `json:"name"`
	Url                   string     `json:"url"`
	UrlResolved           string     `json:"url_resolved"`
	Homepage              string     `json:"homepage"`
	Favicon               string     `json:"favicon"`
	Tags                  string     `json:"tags"`
	Country               string     `json:"country"`
	CountryCode           string     `json:"countrycode"`
	IsoCountryCode        *string    `json:"iso_3166_2,omitempty"`
	State                 string     `json:"state"`
	Language              string     `json:"language"`
	Languagecodes         string     `json:"languagecodes"`
	Votes                 int32      `json:"votes"`
	LastChangeTime        string     `json:"lastchangetime"`
	IsoLastChangeTime     *time.Time `json:"lastchangetime_iso8601"`
	Codec                 string     `json:"codec"`
	Bitrate               uint32     `json:"bitrate"`
	Hls                   uint8      `json:"hls"`
	LastCheckOk           uint8      `json:"lastcheckok"`
	LastCheckTime         string     `json:"lastchecktime"`
	IsoLastCheckTime      *time.Time `json:"lastchecktime_iso8601,omitempty"`
	LastCheckOkTime       string     `json:"lastcheckoktime"`
	IsoLastCheckOkTime    *time.Time `json:"lastcheckoktime_iso8601"`
	LastLocalCheckTime    string     `json:"lastlocalchecktime"`
	IsoLastLocalCheckTime *time.Time `json:"lastlocalchecktime_iso8601,omitempty"`
	ClickTimestamp        string     `json:"clicktimestamp"`
	IsoClickTimestamp     *time.Time `json:"clicktimestamp_iso8601"`
	ClickCount            uint32     `json:"clickcount"`
	ClickTrend            int32      `json:"clicktrend"`
	SslError              uint8      `json:"ssl_error"`
	GeoLat                *float64   `json:"geo_lat,omitempty"`
	GeoLong               *float64   `json:"geo_long,omitempty"`
}

type RadioStations []RadioStation

type UrlResponse struct {
	Ok        bool   `json:"ok"`
	Message   string `json:"message"`
	StationId string `json:"stationuuid"`
	Name      string `json:"name"`
	Url       string `json:"url"`
}
