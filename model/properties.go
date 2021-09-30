package model

const (
	PropUsingMbzIDs = "UsingMbzIDs"
)

type PropertyRepository interface {
	Put(id string, value string) error
	Get(id string) (string, error)
	Delete(id string) error
	DeletePrefixed(prefix string) error
	DefaultGet(id string, defaultValue string) (string, error)
}
