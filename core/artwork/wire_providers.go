package artwork

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtwork,
	NewService,
	GetImageCache,
	NewCacheWarmer,
	NewWorker,
	ProvideImageStore,
)
