package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/spf13/cobra"
)

var (
	playlistID   string
	outputFile   string
	userID       string
	outputFormat string
)

type displayPlaylist struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	OwnerName string `json:"ownerName"`
	OwnerId   string `json:"ownerId"`
	Public    bool   `json:"public"`
}

type displayPlaylists []displayPlaylist

func init() {
	plsCmd.Flags().StringVarP(&playlistID, "playlist", "p", "", "playlist name or ID")
	plsCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default stdout)")
	_ = plsCmd.MarkFlagRequired("playlist")
	rootCmd.AddCommand(plsCmd)

	listCommand.Flags().StringVarP(&userID, "user", "u", "", "username or ID")
	listCommand.Flags().StringVarP(&outputFormat, "format", "f", "csv", "output format [supported values: csv, json]")
	plsCmd.AddCommand(listCommand)
}

var (
	plsCmd = &cobra.Command{
		Use:   "pls",
		Short: "Export playlists",
		Long:  "Export Navidrome playlists to M3U files",
		Run: func(cmd *cobra.Command, args []string) {
			runExporter(cmd.Context())
		},
	}

	listCommand = &cobra.Command{
		Use:   "list",
		Short: "List playlists",
		Run: func(cmd *cobra.Command, args []string) {
			runList(cmd.Context())
		},
	}
)

func runExporter(ctx context.Context) {
	ds, ctx := getAdminContext(ctx)
	playlist, err := ds.Playlist(ctx).GetWithTracks(playlistID, true, false)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Fatal("Error retrieving playlist", "name", playlistID, err)
	}
	if errors.Is(err, model.ErrNotFound) {
		playlists, err := ds.Playlist(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"playlist.name": playlistID}})
		if err != nil {
			log.Fatal("Error retrieving playlist", "name", playlistID, err)
		}
		if len(playlists) > 0 {
			playlist, err = ds.Playlist(ctx).GetWithTracks(playlists[0].ID, true, false)
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

func runList(ctx context.Context) {
	if outputFormat != "csv" && outputFormat != "json" {
		log.Fatal("Invalid output format. Must be one of csv, json", "format", outputFormat)
	}

	ds, ctx := getAdminContext(ctx)
	options := model.QueryOptions{Sort: "owner_name"}

	if userID != "" {
		user, err := getUser(ctx, userID, ds)
		if err != nil {
			log.Fatal(ctx, "Error retrieving user", "username or id", userID)
		}
		options.Filters = squirrel.Eq{"owner_id": user.ID}
	}

	playlists, err := ds.Playlist(ctx).GetAll(options)
	if err != nil {
		log.Fatal(ctx, "Failed to retrieve playlists", err)
	}

	if outputFormat == "csv" {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"playlist id", "playlist name", "owner id", "owner name", "public"})
		for _, playlist := range playlists {
			_ = w.Write([]string{playlist.ID, playlist.Name, playlist.OwnerID, playlist.OwnerName, strconv.FormatBool(playlist.Public)})
		}
		w.Flush()
	} else {
		display := make(displayPlaylists, len(playlists))
		for idx, playlist := range playlists {
			display[idx].Id = playlist.ID
			display[idx].Name = playlist.Name
			display[idx].OwnerId = playlist.OwnerID
			display[idx].OwnerName = playlist.OwnerName
			display[idx].Public = playlist.Public
		}

		j, _ := json.Marshal(display)
		fmt.Printf("%s\n", j)
	}
}
