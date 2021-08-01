package model

import (
	"time"
)

type Share struct {
	ID            string    `structs:"id" json:"id"            orm:"column(id)"`
	Name          string    `structs:"name" json:"name"`
	Description   string    `structs:"description" json:"description"`
	ExpiresAt     time.Time `structs:"expires_at" json:"expiresAt"`
	CreatedAt     time.Time `structs:"created_at" json:"createdAt"`
	LastVisitedAt time.Time `structs:"last_visited_at" json:"lastVisitedAt"`
	ResourceIDs   string    `structs:"resource_ids" json:"resourceIds"   orm:"column(resource_ids)"`
	ResourceType  string    `structs:"resource_type" json:"resourceType"`
	VisitCount    int       `structs:"visit_count" json:"visitCount"`
}

type Shares []Share

type ShareRepository interface {
	Put(s *Share) error
	GetAll(options ...QueryOptions) (Shares, error)
}
