package model

const (
	// TODO Move other prop keys to here
	PropLastScan    = "LastScan"
	PropUsingMbzIDs = "UsingMbzIDs"
)

type PropertyRepository interface {
	Put(id string, value string) error
	Get(id string) (string, error)
	Delete(id string) error
	DefaultGet(id string, defaultValue string) (string, error)
}
