package model

type Transcoding struct {
	ID             string `json:"id"            orm:"column(id)"`
	Name           string `json:"name"`
	TargetFormat   string `json:"targetFormat"`
	Command        string `json:"command"`
	DefaultBitRate int    `json:"defaultBitRate"`
}

type Transcodings []Transcoding

type TranscodingRepository interface {
	Get(id string) (*Transcoding, error)
	CountAll(...QueryOptions) (int64, error)
	Put(*Transcoding) error
	FindByFormat(format string) (*Transcoding, error)
}
