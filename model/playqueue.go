package model

import (
	"time"
)

type PlayQueue struct {
	ID        string     `structs:"id" json:"id"`
	UserID    string     `structs:"user_id" json:"userId"`
	Current   string     `structs:"current" json:"current"`
	Position  int64      `structs:"position" json:"position"`
	ChangedBy string     `structs:"changed_by" json:"changedBy"`
	Items     MediaFiles `structs:"-" json:"items,omitempty"`
	CreatedAt time.Time  `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `structs:"updated_at" json:"updatedAt"`
}

type PlayQueues []PlayQueue

type PlayQueueRepository interface {
	Store(queue *PlayQueue) error
	Retrieve(userId string) (*PlayQueue, error)
}
