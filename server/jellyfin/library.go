package jellyfin

import (
	"context"
	"strconv"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// accessibleLibraryIDs returns the ids of the libraries the current user can access. An
// empty slice only occurs for the edge case of a non-admin user with no assigned libraries;
// filter.ApplyLibraryFilter/ApplyArtistLibraryFilter treat it as a no-op (unrestricted).
func accessibleLibraryIDs(ctx context.Context) []int {
	u, _ := request.UserFrom(ctx)
	return u.Libraries.IDs()
}

// resolveLibraryScope handles ParentId's ambiguity: it can be a library id (browsing a
// UserView) or an entity id (an artist for MusicAlbum queries, an album for Audio queries).
// It's only treated as a library when the user actually has access to it; otherwise
// isLibraryParent is false and callers should fall through to treating parentId as an entity
// id, which safely matches nothing rather than leaking another library's content.
func resolveLibraryScope(ctx context.Context, parentId string) (scopeIDs []int, isLibraryParent bool) {
	if parentId != "" {
		if id, err := strconv.Atoi(parentId); err == nil {
			if u, _ := request.UserFrom(ctx); u.HasLibraryAccess(id) {
				return []int{id}, true
			}
		}
	}
	return accessibleLibraryIDs(ctx), false
}

// libraryView builds the CollectionFolder BaseItemDto Jellyfin clients use to represent a
// library as a top-level browsing node. It's shared by getUserViews (which lists every
// accessible library) and getItem (which resolves a single one), since Finamp fetches a
// UserView's id as a plain item to load the library.
func libraryView(lib model.Library) dto.BaseItemDto {
	return dto.BaseItemDto{
		Id:                dto.EncodeID(strconv.Itoa(lib.ID)),
		Name:              lib.Name,
		Type:              "CollectionFolder",
		CollectionType:    "music",
		IsFolder:          true,
		BackdropImageTags: []string{},
	}
}
