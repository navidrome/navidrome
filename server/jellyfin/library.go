package jellyfin

import (
	"context"

	"github.com/navidrome/navidrome/model/request"
)

// accessibleLibraryIDs returns the ids of the libraries the current user can access. An
// empty result means "no restriction" (e.g. admins), which matches the no-op semantics of
// filter.ApplyLibraryFilter/ApplyArtistLibraryFilter for an empty id list.
func accessibleLibraryIDs(ctx context.Context) []int {
	u, _ := request.UserFrom(ctx)
	return u.Libraries.IDs()
}
