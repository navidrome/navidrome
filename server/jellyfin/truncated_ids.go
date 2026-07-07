package jellyfin

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// truncatedIDLen is what Finamp's saved-queue persistence cuts item ids to (16 bytes, assuming
// Jellyfin GUIDs). No Navidrome id family is 16 chars (nanoid=22, legacy MD5=32, playlist
// UUID=36), so the length alone identifies a truncated id. See README.
const truncatedIDLen = 16

// resolveItemID maps a truncated item id back to the full id via unique-prefix lookup. The id is
// returned unchanged when it isn't truncation-shaped, matches nothing, or is ambiguous.
func (api *Router) resolveItemID(ctx context.Context, id string) string {
	if len(id) != truncatedIDLen {
		return id
	}
	probes := []func() []string{
		func() []string { return idsMatching(api.ds.MediaFile(ctx).GetAll, "media_file.id", id, mediaFileID) },
		func() []string { return idsMatching(api.ds.Album(ctx).GetAll, "album.id", id, albumID) },
		func() []string { return idsMatching(api.ds.Artist(ctx).GetAll, "artist.id", id, artistID) },
		func() []string { return idsMatching(api.ds.Playlist(ctx).GetAll, "playlist.id", id, playlistID) },
	}
	for _, probe := range probes {
		switch ids := probe(); len(ids) {
		case 0:
			continue
		case 1:
			log.Trace(ctx, "Jellyfin API: resolved truncated item id", "truncated", id, "full", ids[0])
			return ids[0]
		default:
			log.Warn(ctx, "Jellyfin API: truncated item id is ambiguous", "truncated", id)
			return id
		}
	}
	return id
}

// idsMatching returns the ids of up to two rows whose id starts with prefix (two is enough to
// detect ambiguity). '\x7f' is above every character the id alphabets use.
func idsMatching[S ~[]T, T any](getAll func(...model.QueryOptions) (S, error), column, prefix string, id func(T) string) []string {
	rows, err := getAll(model.QueryOptions{
		Filters: squirrel.And{squirrel.GtOrEq{column: prefix}, squirrel.Lt{column: prefix + "\x7f"}},
		Max:     2,
	})
	if err != nil {
		return nil
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = id(row)
	}
	return ids
}

func mediaFileID(mf model.MediaFile) string { return mf.ID }
func albumID(al model.Album) string         { return al.ID }
func artistID(ar model.Artist) string       { return ar.ID }
func playlistID(pl model.Playlist) string   { return pl.ID }
