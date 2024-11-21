package scanner

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
)

var (
	ErrAlreadyScanning = errors.New("already scanning")
)

type Scanner interface {
	ScanAll(ctx context.Context, fullRescan bool) error
	Status(context.Context) (*StatusInfo, error)
}

type StatusInfo struct {
	Scanning    bool
	LastScan    time.Time
	Count       uint32
	FolderCount uint32
}

func GetInstance(rootCtx context.Context, ds model.DataStore, cw artwork.CacheWarmer) Scanner {
	if conf.Server.DevExternalScanner {
		return GetExternalInstance(rootCtx)
	}
	return GetLocalInstance(rootCtx, ds, cw)
}

func GetExternalInstance(rootCtx context.Context) Scanner {
	return singleton.GetInstance(func() *scannerClient {
		return &scannerClient{rootCtx: rootCtx}
	})
}

func GetLocalInstance(rootCtx context.Context, ds model.DataStore, cw artwork.CacheWarmer) Scanner {
	return singleton.GetInstance(func() *scanner {
		return &scanner{
			rootCtx: rootCtx,
			ds:      ds,
			cw:      cw,
		}
	})
}
