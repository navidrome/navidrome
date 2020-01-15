package domain

type CheckSumRepository interface {
	Get(id string) (string, error)
	SetData(newSums map[string]string) error
}
