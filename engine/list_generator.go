package engine

import (
	"math/rand"
	"time"

	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
)

// TODO Use Entries instead of Albums
type ListGenerator interface {
	GetNewest(offset int, size int) (*domain.Albums, error)
	GetRecent(offset int, size int) (*domain.Albums, error)
	GetFrequent(offset int, size int) (*domain.Albums, error)
	GetHighest(offset int, size int) (*domain.Albums, error)
	GetRandom(offset int, size int) (*domain.Albums, error)
	GetStarred() (*Entries, error)
	GetNowPlaying() (*Entries, error)
}

func NewListGenerator(alr domain.AlbumRepository, mfr domain.MediaFileRepository, npr NowPlayingRepository) ListGenerator {
	return listGenerator{alr, mfr, npr}
}

type listGenerator struct {
	albumRepo    domain.AlbumRepository
	mfRepository domain.MediaFileRepository
	npRepo       NowPlayingRepository
}

func (g listGenerator) query(qo domain.QueryOptions, offset int, size int) (*domain.Albums, error) {
	qo.Offset = offset
	qo.Size = size
	return g.albumRepo.GetAll(qo)
}

func (g listGenerator) GetNewest(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "CreatedAt", Desc: true, Alpha: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetRecent(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "PlayDate", Desc: true, Alpha: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetFrequent(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "PlayCount", Desc: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetHighest(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "Rating", Desc: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetRandom(offset int, size int) (*domain.Albums, error) {
	ids, err := g.albumRepo.GetAllIds()
	if err != nil {
		return nil, err
	}
	size = utils.MinInt(size, len(*ids))
	perm := rand.Perm(size)
	r := make(domain.Albums, size)

	for i := 0; i < size; i++ {
		v := perm[i]
		al, err := g.albumRepo.Get((*ids)[v])
		if err != nil {
			return nil, err
		}
		r[i] = *al
	}
	return &r, nil
}

func (g listGenerator) GetStarred() (*Entries, error) {
	albums, err := g.albumRepo.GetStarred(domain.QueryOptions{})
	if err != nil {
		return nil, err
	}
	entries := make(Entries, len(*albums))

	for i, al := range *albums {
		entries[i] = FromAlbum(&al)
	}

	return &entries, nil
}

func (g listGenerator) GetNowPlaying() (*Entries, error) {
	npInfo, err := g.npRepo.GetAll()
	if err != nil {
		return nil, err
	}
	entries := make(Entries, len(*npInfo))
	for i, np := range *npInfo {
		mf, err := g.mfRepository.Get(np.TrackId)
		if err != nil {
			return nil, err
		}
		entries[i] = FromMediaFile(mf)
		entries[i].UserName = np.Username
		entries[i].MinutesAgo = int(time.Now().Sub(np.Start).Minutes())
		entries[i].PlayerId = np.PlayerId
		entries[i].PlayerName = np.PlayerName

	}
	return &entries, nil
}
