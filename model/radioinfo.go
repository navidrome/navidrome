package model

type BaseRadioInfo struct {
	Tags        string `structs:"tags"         json:"tags,omitempty"`
	Country     string `structs:"country"      json:"country,omitempty"`
	CountryCode string `structs:"country_code" json:"countryCode,omitempty"`
	Codec       string `structs:"codec"        json:"codec,omitempty"`
	Bitrate     uint32 `structs:"bitrate"      json:"bitrate,omitempty"`
}

type RadioInfo struct {
	BaseRadioInfo `structs:"-"`
	ID            string `structs:"id"           json:"id" orm:"pk;column(id)"`
	Name          string `structs:"name"         json:"name"`
	Url           string `structs:"url"          json:"url"`
	Homepage      string `structs:"homepage"     json:"homepage"`
	Favicon       string `structs:"favicon"      json:"favicon"`
	Existing      bool   `structs:"existing"       json:"existing"`
}

type RadioInfos []RadioInfo

type RadioInfoRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Get(id string) (*RadioInfo, error)
	GetAll(options ...QueryOptions) (RadioInfos, error)
	Insert(m *RadioInfo) error
	Purge() error
}
