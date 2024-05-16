package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/spf13/cobra"
)

var mbzidNoScan bool
var mbzidNoConfirm bool

var mbzIdCmd = &cobra.Command{
	Use:   "use_mbzid",
	Short: "Use MusicBrainz IDs",
	Long:  "Convert Navidrome's database to use MusicBrainz IDs",
	Run: func(cmd *cobra.Command, args []string) {
		db.Init()
		if err := convertToMbzIDs(cmd.Context()); err != nil {
			log.Error("Error handling MusicBrainz cataloging. Aborting", err)
			os.Exit(1)
			return
		}
	},
}

func init() {
	mbzIdCmd.Flags().BoolVar(&mbzidNoScan, "no-scan", false, `don't re-scan afterwards.
WARNING: Your database will be in an inconsistent state unless a full rescan is completed.`)
	mbzIdCmd.Flags().BoolVar(&mbzidNoConfirm, "no-confirm", false, "don't ask for confirmation")
	rootCmd.AddCommand(mbzIdCmd)
}

func warnMbzMigration(dur time.Duration) bool {
	log.Warn("About to convert database to use MusicBrainz metadata. This CANNOT be undone.")
	log.Warn(fmt.Sprintf("If this isn't intentional, press ^C NOW. Will begin in %s...", dur))

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)

	defer signal.Stop(sc)

	select {
	case <-sc:
		return false
	case <-time.After(dur):
		return true
	}
}

type deleteManyable interface {
	DeleteMany(ids ...string) error
}

func deleteManyIDs(repo deleteManyable, ids map[string]bool) error {
	s := make([]string, 0, len(ids))
	for id := range ids {
		s = append(s, id)
	}

	return slice.RangeByChunks(s, 100, func(s []string) error {
		return repo.DeleteMany(s...)
	})
}

func migrateUserPlaylists(ctx context.Context, ds model.DataStore, user model.User, ndIdToMbz map[string]*model.MediaFile) error {
	var err error

	repo := ds.Playlist(request.WithUser(ctx, user))
	playlists, err := repo.GetAll()
	if err != nil {
		return err
	}

	for _, playlist := range playlists {
		newPlaylist, err2 := repo.GetWithTracks(playlist.ID, false)
		if err2 != nil {
			return err2
		}

		for i, track := range newPlaylist.Tracks {
			if newTrack, found := ndIdToMbz[track.MediaFileID]; found {
				newPlaylist.Tracks[i].MediaFileID = newTrack.ID
				newPlaylist.Tracks[i].MediaFile.ID = newTrack.ID
			}
		}

		if err2 = repo.Put(newPlaylist); err2 != nil {
			return err2
		}
	}
	return nil
}

func migrateUserPlayQueue(ctx context.Context, ds model.DataStore, user model.User, ndIdToMbz map[string]*model.MediaFile) error {
	repo := ds.PlayQueue(request.WithUser(ctx, user))
	playQueue, err := repo.Retrieve(user.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil
		}

		return err
	}

	if newTrack, found := ndIdToMbz[playQueue.Current]; found {
		playQueue.Current = newTrack.ID
	}

	for i, item := range playQueue.Items {
		if newTrack, found := ndIdToMbz[item.ID]; found {
			playQueue.Items[i].ID = newTrack.ID
		}
	}

	return repo.Store(playQueue)
}

func fillArtists(repo model.ArtistRepository, newArtists map[string]*model.Artist) error {
	artists, err := repo.GetAll()
	if err != nil {
		return err
	}

	for _, artist := range artists {
		if newArtist, ok := newArtists[artist.MbzArtistID]; ok {
			tmp := *newArtist
			*newArtist = artist
			newArtist.ID = tmp.ID
			newArtist.MbzArtistID = tmp.MbzArtistID
		}
	}

	return nil
}

func fillAlbums(repo model.AlbumRepository, newAlbums map[string]*model.Album) error {
	albums, err := repo.GetAll()
	if err != nil {
		return err
	}

	for _, album := range albums {
		if newAlbum, ok := newAlbums[album.MbzAlbumID]; ok {
			tmp := *newAlbum
			*newAlbum = album
			newAlbum.ID = tmp.ID
			newAlbum.ArtistID = tmp.ArtistID
			newAlbum.AlbumArtistID = tmp.AlbumArtistID
			newAlbum.MbzAlbumID = tmp.MbzAlbumID
			newAlbum.MbzAlbumArtistID = tmp.MbzAlbumArtistID
			newAlbum.AllArtistIDs = "" // Nuking this, the rescan will fix it
		}
	}

	return nil
}

// Migrate all the database entities to use MusicBrainz IDs.
// Uses the Mbz* fields in model.MediaFile to define the relationships, ignoring
// the Navidrome ones.
func migrateEverything(ctx context.Context, ds model.DataStore) error {
	artistRepo := ds.Artist(ctx)
	albumRepo := ds.Album(ctx)
	mfRepo := ds.MediaFile(ctx)

	log.Info("Pass 1: Rebuild hierarchy")

	mediaFiles, err := mfRepo.GetAll()
	if err != nil {
		return err
	}

	newMediaFiles := make(map[string]*model.MediaFile, len(mediaFiles))
	newArtists := map[string]*model.Artist{}
	newAlbums := map[string]*model.Album{}

	oldMediaFiles := make(map[string]bool, len(mediaFiles))
	oldArtists := map[string]bool{}
	oldAlbums := map[string]bool{}

	oldToNewMF := make(map[string]*model.MediaFile, len(mediaFiles)) // For play queue/playlist remapping
	newToOldMF := make(map[string]string, len(mediaFiles))           // For mediafile annotations
	newToOldAlbum := map[string]string{}                             // For album annotations
	newToOldArtist := map[string]string{}                            // For artist annotations

	for _, mf := range mediaFiles {
		// Don't touch partial files. The final rescan should take care of them.
		if mf.MbzReleaseTrackID == "" || mf.MbzAlbumID == "" || mf.MbzArtistID == "" || mf.MbzAlbumArtistID == "" {
			continue
		}

		newID := mf.MbzReleaseTrackID

		if _, ok := newMediaFiles[newID]; !ok {
			newMediaFile := &model.MediaFile{}
			*newMediaFile = mf

			newMediaFile.ID = newID
			newMediaFile.AlbumID = mf.MbzAlbumID
			newMediaFile.ArtistID = mf.MbzArtistID
			newMediaFile.AlbumArtistID = mf.MbzAlbumArtistID
			newMediaFiles[newID] = newMediaFile

			oldToNewMF[mf.ID] = newMediaFile
			newToOldMF[newID] = mf.ID
			oldMediaFiles[mf.ID] = true
		}

		if _, ok := newArtists[mf.MbzArtistID]; !ok {
			newArtists[mf.MbzArtistID] = &model.Artist{ID: mf.MbzArtistID, MbzArtistID: mf.MbzArtistID}
			newToOldArtist[mf.MbzArtistID] = mf.ArtistID
			oldArtists[mf.ArtistID] = true
		}

		if _, ok := newArtists[mf.MbzAlbumArtistID]; !ok {
			newArtists[mf.MbzAlbumArtistID] = &model.Artist{ID: mf.MbzAlbumArtistID, MbzArtistID: mf.MbzAlbumArtistID}
			newToOldArtist[mf.MbzAlbumArtistID] = mf.AlbumArtistID
			oldArtists[mf.AlbumArtistID] = true
		}

		if _, ok := newAlbums[mf.MbzAlbumID]; !ok {
			newAlbums[mf.MbzAlbumID] = &model.Album{
				ID:               mf.MbzAlbumID,
				ArtistID:         mf.MbzArtistID,
				AlbumArtistID:    mf.MbzAlbumArtistID,
				MbzAlbumID:       mf.MbzAlbumID,
				MbzAlbumArtistID: mf.MbzAlbumArtistID,
			}
			newToOldAlbum[mf.MbzAlbumID] = mf.AlbumID
			oldAlbums[mf.AlbumID] = true
		}
	}

	// Attempt to salvage some artist/album information.
	// These parts are completely optional, as all the information will be recovered by the final rescan.
	if err = fillAlbums(albumRepo, newAlbums); err != nil {
		return err
	}

	if err = fillArtists(artistRepo, newArtists); err != nil {
		return err
	}

	log.Info("Pass 2: Add new artists", "count", len(newArtists))
	for _, artist := range newArtists {
		if err = artistRepo.Put(artist); err != nil {
			return err
		}

		if err = artistRepo.CopyAnnotation(newToOldArtist[artist.ID], artist.ID); err != nil {
			return err
		}
	}

	log.Info("Pass 3: Add new albums", "count", len(newAlbums))
	for _, album := range newAlbums {
		if err = albumRepo.Put(album); err != nil {
			return err
		}

		if err = albumRepo.CopyAnnotation(newToOldAlbum[album.ID], album.ID); err != nil {
			return err
		}
	}

	log.Info("Pass 4: Add new tracks", "count", len(newMediaFiles))
	for _, mf := range newMediaFiles {
		if err = mfRepo.Put(mf); err != nil {
			return err
		}

		if err = mfRepo.CopyAnnotation(newToOldMF[mf.ID], mf.ID); err != nil {
			return err
		}
	}

	// Playlists and Play queues require a user in the context
	users, err := ds.User(ctx).GetAll()
	if err != nil {
		return err
	}

	log.Info("Pass 5: Update playlist references")
	for _, user := range users {
		if err = migrateUserPlaylists(ctx, ds, user, oldToNewMF); err != nil {
			return err
		}
	}

	log.Info("Pass 6: Update play queue references", "count", len(users))
	for _, user := range users {
		if err = migrateUserPlayQueue(ctx, ds, user, oldToNewMF); err != nil {
			return err
		}
	}

	log.Info("Pass 7: Cleanup leftover tracks", "count", len(oldMediaFiles))
	if err = deleteManyIDs(mfRepo, oldMediaFiles); err != nil {
		return err
	}

	log.Info("Pass 8: Cleanup leftover albums", "count", len(oldAlbums))
	if err = deleteManyIDs(albumRepo, oldAlbums); err != nil {
		return err
	}

	log.Info("Pass 9: Cleanup leftover artists", "count", len(oldArtists))
	if err = deleteManyIDs(artistRepo, oldArtists); err != nil {
		return err
	}

	return nil
}

func convertToMbzIDs(ctx context.Context) error {
	var err error

	ds := persistence.New(db.Db())

	alreadyDone := false

	err = ds.WithTx(func(tx model.DataStore) error {
		props := tx.Property(ctx)

		useMbzIDs, err := props.DefaultGetBool(model.PropUsingMbzIDs, false)
		if err != nil {
			return err
		}

		// Nothing to do
		if useMbzIDs {
			alreadyDone = true
			return nil
		}

		if !mbzidNoConfirm && !warnMbzMigration(10*time.Second) {
			return errors.New("user aborted")
		}

		if err := migrateEverything(ctx, tx); err != nil {
			return err
		}

		if err = props.Put(model.PropUsingMbzIDs, "true"); err != nil {
			return err
		}

		return tx.Library(ctx).UpdateLastScan(1, time.Unix(0, 0))
	})

	if err != nil {
		return err
	}

	if alreadyDone {
		log.Info("Migration already done.")
		return nil
	}

	if mbzidNoScan {
		log.Info("Skipping post-migration scan by request.")
		return nil
	}

	fullRescan = true
	runScanner()
	return nil
}
