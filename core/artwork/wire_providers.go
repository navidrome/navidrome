package artwork

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewService,
	GetImageCache,
	NewWorker,
	ProvideImageStore,
)
