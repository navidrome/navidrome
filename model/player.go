package model

import (
	"time"
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
	Get(id string) (*Player, error)
	FindMatch(userId, client, userAgent string) (*Player, error)
	Put(p *Player) error
	CountAll(...QueryOptions) (int64, error)
	CountByClient(...QueryOptions) (map[string]int64, error)
	// TODO: Add CountAll method. Useful at least for metrics.
}
