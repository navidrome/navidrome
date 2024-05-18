package model

import (
	"time"
)

type Player struct {
	ID              string    `structs:"id" json:"id"`
	Name            string    `structs:"name" json:"name"`
	UserAgent       string    `structs:"user_agent" json:"userAgent"`
	UserName        string    `structs:"user_name" json:"userName"`
	Client          string    `structs:"client" json:"client"`
	IPAddress       string    `structs:"ip_address" json:"ipAddress"`
	LastSeen        time.Time `structs:"last_seen" json:"lastSeen"`
	TranscodingId   string    `structs:"transcoding_id" json:"transcodingId"`
	MaxBitRate      int       `structs:"max_bit_rate" json:"maxBitRate"`
	ReportRealPath  bool      `structs:"report_real_path" json:"reportRealPath"`
	ScrobbleEnabled bool      `structs:"scrobble_enabled" json:"scrobbleEnabled"`
}

type Players []Player

type PlayerRepository interface {
	Get(id string) (*Player, error)
	FindMatch(userName, client, typ string) (*Player, error)
	Put(p *Player) error
	// TODO: Add CountAll method. Useful at least for metrics.
}
