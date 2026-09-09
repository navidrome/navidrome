package scanner

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/model"
)

// PIDConfChanged reports whether any library's effective PID specs differ from
// the specs recorded by its last completed scan.
func PIDConfChanged(ctx context.Context, ds model.DataStore) (bool, error) {
	libs, err := ds.Library(ctx).GetAll()
	if err != nil {
		return false, err
	}
	for _, lib := range libs {
		if !strings.EqualFold(lib.ScannedPIDAlbum, lib.EffectivePIDAlbum()) ||
			!strings.EqualFold(lib.ScannedPIDTrack, lib.EffectivePIDTrack()) {
			return true, nil
		}
	}
	return false, nil
}
