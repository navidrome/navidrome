package scanner

import (
	"time"
)

type Track struct {
	Id         string
	Path       string
	Album      string
	Artist     string
	Title      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}