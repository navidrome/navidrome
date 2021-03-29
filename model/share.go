package model

import (
	"time"
)

type Share struct {
	ID            string    `json:"id"          orm:"column(id)"`
	Url           string    `json:"url"`
	Description   string    `json:"description"`
	ExpiresAt     time.Time `json:"expires"`
	CreatedAt     time.Time `json:"created"`
	LastVisitedAt time.Time `json:"lastVisited"`
	ResourceID    string    `json:"resourceID"`
	ResourceType  string    `json:"resourceType"`
	VisitCount    string    `json:"visitCount"`
}

type Shares []Share

type ShareRepository interface {
	Put(s *Share) (*Share, error)
	GetAll(options ...QueryOptions) (Shares, error)
	Update(s *Share) error
	Delete(id string) error
}
