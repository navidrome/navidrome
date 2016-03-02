package domain

type Property struct {
	Id    string
	Value string
}

type PropertyRepository interface {
	Put(id string, value string) error
	Get(id string) (string, error)
	DefaultGet(id string, defaultValue string) (string, error)
}
