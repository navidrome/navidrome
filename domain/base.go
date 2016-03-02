package domain

type BaseRepository interface {
	NewId(fields ...string) string
	CountAll() (int, error)

}

