package jellyfin

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// accessibleLibraryIDs returns the ids of the libraries the current user can access. An empty
// slice (non-admin with no libraries) is treated as a no-op/unrestricted by the library filters.
func accessibleLibraryIDs(ctx context.Context) []int {
	u, _ := request.UserFrom(ctx)
	return u.Libraries.IDs()
}

// resolveLibraryScope handles ParentId's ambiguity: a library id (browsing a UserView) or an
// entity id (artist/album). It's treated as a library only when the user has access; otherwise
// isLibraryParent is false and callers fall through to entity-id handling.
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

// parentIDScope resolves the request's ParentId param to a library scope (see resolveLibraryScope).
func parentIDScope(ctx context.Context, r *http.Request) (scopeIDs []int, isLibraryParent bool) {
	return resolveLibraryScope(ctx, dto.DecodeID(req.Params(r).StringOr("parentid", "")))
}

// libraryScopeFilter restricts a tag query to the given library scope. Empty scope means
// unrestricted (see accessibleLibraryIDs), so it returns nil rather than an impossible IN ().
func libraryScopeFilter(scope []int) squirrel.Sqlizer {
	if len(scope) == 0 {
		return nil
	}
	return squirrel.Eq{"library_tag.library_id": scope}
}

// libraryView builds the CollectionFolder BaseItemDto representing a library as a top-level node.
// Shared by getUserViews and getItem, since Finamp fetches a UserView's id as a plain item.
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
