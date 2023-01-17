package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/spf13/cobra"
)

var (
	playlistID string
	outputFile string
)

func init() {
	plsCmd.Flags().StringVarP(&playlistID, "playlist", "p", "", "playlist name or ID")
	plsCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default stdout)")
	_ = plsCmd.MarkFlagRequired("playlist")
	rootCmd.AddCommand(plsCmd)
}

var plsCmd = &cobra.Command{
	Use:   "pls",
	Short: "Export playlists",
	Long:  "Export Navidrome playlists to M3U files",
	Run: func(cmd *cobra.Command, args []string) {
		runExporter()
	},
}

func runExporter() {
	sqlDB := db.Db()
	ds := persistence.New(sqlDB)
	ctx := auth.WithAdminUser(context.Background(), ds)
	playlist, err := ds.Playlist(ctx).GetWithTracks(playlistID, true)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Fatal("Error retrieving playlist", "name", playlistID, err)
	}
	if errors.Is(err, model.ErrNotFound) {
		playlists, err := ds.Playlist(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"playlist.name": playlistID}})
		if err != nil {
			log.Fatal("Error retrieving playlist", "name", playlistID, err)
		}
		if len(playlists) > 0 {
			playlist, err = ds.Playlist(ctx).GetWithTracks(playlists[0].ID, true)
			if err != nil {
				log.Fatal("Error retrieving playlist", "name", playlistID, err)
			}
		}
	}
	if playlist == nil {
		log.Fatal("Playlist not found", "name", playlistID)
	}
	pls := playlist.ToM3U8()
	if outputFile == "-" || outputFile == "" {
		println(pls)
		return
	}

	err = os.WriteFile(outputFile, []byte(pls), 0600)
	if err != nil {
		log.Fatal("Error writing to the output file", "file", outputFile, err)
	}
}
