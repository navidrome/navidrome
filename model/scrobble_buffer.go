package model

import (
	"context"
	"time"
)

type ScrobbleEntry struct {
	ID          string
	Service     string
	UserID      string
	PlayTime    time.Time
	EnqueueTime time.Time
	MediaFileID string
	MediaFile
}

type ScrobbleEntries []ScrobbleEntry

type ScrobbleBufferRepository interface {
	UserIDs(ctx context.Context, service string) ([]string, error)
	Enqueue(ctx context.Context, service, userId, mediaFileId string, playTime time.Time) error
	Next(ctx context.Context, service string, userId string) (*ScrobbleEntry, error)
	Dequeue(ctx context.Context, entry *ScrobbleEntry) error
	Length(ctx context.Context) (int64, error)
	Discard(ctx context.Context, service string) error
}
