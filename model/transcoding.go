package model

import (
	"context"

	"github.com/deluan/rest"
)

type Transcoding struct {
	ID             string `structs:"id" json:"id"`
	Name           string `structs:"name" json:"name"`
	TargetFormat   string `structs:"target_format" json:"targetFormat"`
	Command        string `structs:"command" json:"command"`
	DefaultBitRate int    `structs:"default_bit_rate" json:"defaultBitRate"`
}

type Transcodings []Transcoding

type TranscodingRepository interface {
	rest.Repository[Transcoding]
	rest.Persistable[Transcoding]
	Get(ctx context.Context, id string) (*Transcoding, error)
	CountAll(ctx context.Context, options ...QueryOptions) (int64, error)
	Put(ctx context.Context, t *Transcoding) error
	FindByFormat(ctx context.Context, format string) (*Transcoding, error)
}
