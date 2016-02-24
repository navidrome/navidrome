package models

import (
	"time"
)

type MediaFile struct {
	Id         string
	Path       string
	Album      string
	Artist     string
	Title      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}