package model

import (
	"time"
)

type Share struct {
	ID            string    `json:"id"            orm:"column(id)"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ExpiresAt     time.Time `json:"expiresAt"`
	CreatedAt     time.Time `json:"createdAt"`
	LastVisitedAt time.Time `json:"lastVisitedAt"`
	ResourceIDs   string    `json:"resourceIds"   orm:"column(resource_ids)"`
	ResourceType  string    `json:"resourceType"`
	VisitCount    int       `json:"visitCount"`
}

type Shares []Share

type ShareRepository interface {
	Put(s *Share) error
	GetAll(options ...QueryOptions) (Shares, error)
}
