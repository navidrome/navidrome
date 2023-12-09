package model

type Transcoding struct {
	ID             string `structs:"id" json:"id"`
	Name           string `structs:"name" json:"name"`
	TargetFormat   string `structs:"target_format" json:"targetFormat"`
	Command        string `structs:"command" json:"command"`
	DefaultBitRate int    `structs:"default_bit_rate" json:"defaultBitRate"`
}

type Transcodings []Transcoding

type TranscodingRepository interface {
	Get(id string) (*Transcoding, error)
	CountAll(...QueryOptions) (int64, error)
	Put(*Transcoding) error
	FindByFormat(format string) (*Transcoding, error)
}
