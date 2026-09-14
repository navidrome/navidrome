package model

import (
	"context"
	"time"

	"github.com/deluan/rest"
)

type Player struct {
	Username string `structs:"-" json:"userName"`

	ID              string    `structs:"id" json:"id"`
	Name            string    `structs:"name" json:"name"`
	UserAgent       string    `structs:"user_agent" json:"userAgent"`
	UserId          string    `structs:"user_id" json:"userId"`
	Client          string    `structs:"client" json:"client"`
	IP              string    `structs:"ip" json:"ip"`
	LastSeen        time.Time `structs:"last_seen" json:"lastSeen"`
	TranscodingId   string    `structs:"transcoding_id" json:"transcodingId"`
	MaxBitRate      int       `structs:"max_bit_rate" json:"maxBitRate"`
	ReportRealPath  bool      `structs:"report_real_path" json:"reportRealPath"`
	ScrobbleEnabled bool      `structs:"scrobble_enabled" json:"scrobbleEnabled"`
}

type Players []Player

type PlayerRepository interface {
	rest.Repository[Player]
	rest.Persistable[Player]
	Get(ctx context.Context, id string) (*Player, error)
	FindMatch(ctx context.Context, userId, client, userAgent string) (*Player, error)
	Put(ctx context.Context, p *Player) error
	CountAll(ctx context.Context, options ...QueryOptions) (int64, error)
	CountByClient(ctx context.Context, options ...QueryOptions) (map[string]int64, error)
}
