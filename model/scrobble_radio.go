package model

import "time"

type ScrobleRadioEntry struct {
	Artist      string
	Title       string
	Service     string
	UserID      string `structs:"user_id" orm:"column(user_id)"`
	PlayTime    time.Time
	EnqueueTime time.Time
}

type ScrobbleRadioEntries []ScrobleRadioEntry

type ScrobbleRadioRepository interface {
	UserIDs(service string) ([]string, error)
	Enqueue(service, userId, artist, title string, playTime time.Time) error
	Next(service string, userId string) (*ScrobleRadioEntry, error)
	Dequeue(entry *ScrobleRadioEntry) error
	Length() (int64, error)
}
