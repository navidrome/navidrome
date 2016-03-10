package engine

import (
	"errors"
	"time"
)

type Child struct {
	Id          string
	Title       string
	IsDir       bool
	Parent      string
	Album       string
	Year        int
	Artist      string
	Genre       string
	CoverArt    string
	Starred     time.Time
	Track       int
	Duration    int
	Size        string
	Suffix      string
	BitRate     int
	ContentType string
}

var (
	ErrDataNotFound = errors.New("Data Not Found")
)
