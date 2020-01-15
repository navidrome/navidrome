package model

type ChecksumMap map[string]string

type ChecksumRepository interface {
	GetData() (ChecksumMap, error)
	SetData(newSums ChecksumMap) error
}
