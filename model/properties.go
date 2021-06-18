package model

const (
	PropLastScan = "LastScan"
)

type Property struct {
	ID    string
	Value string
}

type PropertyRepository interface {
	Put(id string, value string) error
	Get(id string) (string, error)
	Delete(id string) error
	DefaultGet(id string, defaultValue string) (string, error)
}
