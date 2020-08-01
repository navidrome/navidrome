package model

import (
	"time"
)

type PlayQueue struct {
	ID        string     `json:"id"          orm:"column(id)"`
	UserID    string     `json:"userId"      orm:"column(user_id)"`
	Current   string     `json:"current"`
	Position  int64      `json:"position"`
	ChangedBy string     `json:"changedBy"`
	Items     MediaFiles `json:"items,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type PlayQueues []PlayQueue

type PlayQueueRepository interface {
	Store(queue *PlayQueue) error
	Retrieve(userId string) (*PlayQueue, error)
}
