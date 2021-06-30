package model

import (
	"time"
)

type Player struct {
	ID              string    `json:"id"            orm:"column(id)"`
	Name            string    `json:"name"`
	UserAgent       string    `json:"userAgent"`
	UserName        string    `json:"userName"`
	Client          string    `json:"client"`
	IPAddress       string    `json:"ipAddress"`
	LastSeen        time.Time `json:"lastSeen"`
	TranscodingId   string    `json:"transcodingId"`
	MaxBitRate      int       `json:"maxBitRate"`
	ReportRealPath  bool      `json:"reportRealPath"`
	ScrobbleEnabled bool      `json:"scrobbleEnabled"`
}

type Players []Player

type PlayerRepository interface {
	Get(id string) (*Player, error)
	FindMatch(userName, client, typ string) (*Player, error)
	Put(p *Player) error
}
