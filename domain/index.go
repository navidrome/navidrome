package domain

import "github.com/deluan/gosonic/utils"

type ArtistInfo struct {
	ArtistId string
	Artist   string
}

type ArtistIndex struct {
	Id      string
	Artists ArtistInfos
}

type ArtistInfos []ArtistInfo
func (a ArtistInfos) Len() int { return len(a) }
func (a ArtistInfos) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ArtistInfos) Less(i, j int) bool {
	return utils.NoArticle(a[i].Artist) < utils.NoArticle(a[j].Artist)
}

type ArtistIndexes []ArtistIndex

type ArtistIndexRepository interface {
	BaseRepository
	Put(m *ArtistIndex) error
	Get(id string) (*ArtistIndex, error)
	GetAll() (ArtistIndexes, error)
}
