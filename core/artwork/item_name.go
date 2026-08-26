package artwork

import (
	"context"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/model"
)

// ItemName resolves a kind+id to the entity's display name, and errors when the item
// does not exist. Callers use it to reject ids that would otherwise orphan a queue row.
func ItemName(ctx context.Context, ds model.DataStore, kind model.Kind, id string) (string, error) {
	switch kind {
	case model.KindArtistArtwork:
		ar, err := ds.Artist(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return ar.Name, nil
	case model.KindAlbumArtwork:
		al, err := ds.Album(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return al.Name, nil
	case model.KindPlaylistArtwork:
		pls, err := ds.Playlist(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return pls.Name, nil
	case model.KindRadioArtwork:
		rd, err := ds.Radio(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return rd.Name, nil
	case model.KindMediaFileArtwork:
		mf, err := ds.MediaFile(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return mf.Title, nil
	case model.KindDiscArtwork:
		return discArtworkName(ctx, ds, id)
	}
	return "", fmt.Errorf("unsupported kind %q", kind.Prefix())
}

func discArtworkName(ctx context.Context, ds model.DataStore, id string) (string, error) {
	albumID, discNumber, err := model.ParseDiscArtworkID(id)
	if err != nil {
		return "", err
	}
	al, err := ds.Album(ctx).Get(albumID)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s (disc %d)", al.Name, discNumber)
	// The subtitle is itself a DiscArtPriority candidate, so name it where the chain can be read against it.
	if subtitle := strings.TrimSpace(al.Discs[discNumber]); subtitle != "" {
		name += ": " + subtitle
	}
	return name, nil
}
