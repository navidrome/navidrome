package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/ioutils"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
	"github.com/spf13/cobra"
)

var (
	playlistID   string
	outputFile   string
	userID       string
	outputFormat string
	syncFlag     bool
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

	exportCommand.Flags().StringVarP(&playlistID, "playlist", "p", "", "playlist name or ID")
	exportCommand.Flags().StringVarP(&outputFile, "output", "o", "", "output directory")
	exportCommand.Flags().StringVarP(&userID, "user", "u", "", "username or ID")
	plsCmd.AddCommand(exportCommand)

	importCommand.Flags().StringVarP(&userID, "user", "u", "", "owner username or ID (default: first admin)")
	importCommand.Flags().BoolVar(&syncFlag, "sync", false, "mark imported playlists as synced")
	plsCmd.AddCommand(importCommand)
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

	exportCommand = &cobra.Command{
		Use:   "export",
		Short: "Export playlists to M3U files",
		Long:  "Export one or more Navidrome playlists to M3U files",
		Run: func(cmd *cobra.Command, args []string) {
			runExport(cmd.Context())
		},
	}

	importCommand = &cobra.Command{
		Use:   "import [files...]",
		Short: "Import M3U playlists",
		Long:  "Import one or more M3U files as Navidrome playlists",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runImport(cmd.Context(), args)
		},
	}
)

func fetchPlaylists(ctx context.Context, ds model.DataStore, sort string) model.Playlists {
	options := model.QueryOptions{Sort: sort}
	if userID != "" {
		user, err := getUser(ctx, userID, ds)
		if err != nil {
			log.Fatal(ctx, "Error retrieving user", "username or id", userID)
		}
		options.Filters = squirrel.Eq{"owner_id": user.ID}
	}
	pls, err := ds.Playlist(ctx).GetAll(options)
	if err != nil {
		log.Fatal(ctx, "Failed to retrieve playlists", err)
	}
	return pls
}

func findPlaylist(ctx context.Context, ds model.DataStore, nameOrID string) *model.Playlist {
	playlist, err := ds.Playlist(ctx).GetWithTracks(nameOrID, true, false)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Fatal("Error retrieving playlist", "name", nameOrID, err)
	}
	if errors.Is(err, model.ErrNotFound) {
		playlists, err := ds.Playlist(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"playlist.name": nameOrID}})
		if err != nil {
			log.Fatal("Error retrieving playlist", "name", nameOrID, err)
		}
		if len(playlists) > 0 {
			playlist, err = ds.Playlist(ctx).GetWithTracks(playlists[0].ID, true, false)
			if err != nil {
				log.Fatal("Error retrieving playlist", "name", nameOrID, err)
			}
		}
	}
	if playlist == nil {
		log.Fatal("Playlist not found", "name", nameOrID)
	}
	return playlist
}

func runExporter(ctx context.Context) {
	ds, ctx := getAdminContext(ctx)
	playlist := findPlaylist(ctx, ds, playlistID)
	pls := playlist.ToM3U8()
	if outputFile == "-" || outputFile == "" {
		println(pls)
		return
	}
	err := os.WriteFile(outputFile, []byte(pls), 0600)
	if err != nil {
		log.Fatal("Error writing to the output file", "file", outputFile, err)
	}
}

func runExport(ctx context.Context) {
	ds, ctx := getAdminContext(ctx)

	if playlistID != "" && outputFile == "" {
		playlist := findPlaylist(ctx, ds, playlistID)
		println(playlist.ToM3U8())
		return
	}

	if outputFile == "" {
		log.Fatal("Output directory (-o) is required for bulk export or when filtering by user")
	}

	info, err := os.Stat(outputFile)
	if err != nil || !info.IsDir() {
		log.Fatal("Output path must be an existing directory", "path", outputFile)
	}

	if playlistID != "" {
		pls := findPlaylist(ctx, ds, playlistID)
		filename := str.SanitizeFilename(pls.Name) + ".m3u"
		path := filepath.Join(outputFile, filename)
		err := os.WriteFile(path, []byte(pls.ToM3U8()), 0600)
		if err != nil {
			log.Fatal("Error writing playlist", "file", path, err)
		}
		fmt.Printf("Exported \"%s\" to %s\n", pls.Name, path)
		return
	}

	allPls := fetchPlaylists(ctx, ds, "name")

	nameCounts := make(map[string]int)
	for _, pls := range allPls {
		nameCounts[str.SanitizeFilename(pls.Name)]++
	}

	exported := 0
	for _, pls := range allPls {
		plsWithTracks, err := ds.Playlist(ctx).GetWithTracks(pls.ID, true, false)
		if err != nil {
			log.Error("Error loading playlist tracks", "playlist", pls.Name, err)
			continue
		}

		sanitized := str.SanitizeFilename(pls.Name)
		filename := sanitized + ".m3u"
		if nameCounts[sanitized] > 1 {
			shortID := pls.ID
			if len(shortID) > 6 {
				shortID = shortID[:6]
			}
			filename = sanitized + "_" + shortID + ".m3u"
		}

		path := filepath.Join(outputFile, filename)
		err = os.WriteFile(path, []byte(plsWithTracks.ToM3U8()), 0600)
		if err != nil {
			log.Error("Error writing playlist", "file", path, err)
			continue
		}
		fmt.Printf("Exported \"%s\" to %s\n", pls.Name, path)
		exported++
	}
	fmt.Printf("\nExported %d playlists to %s\n", exported, outputFile)
}

func runList(ctx context.Context) {
	if outputFormat != "csv" && outputFormat != "json" {
		log.Fatal("Invalid output format. Must be one of csv, json", "format", outputFormat)
	}

	ds, ctx := getAdminContext(ctx)
	allPls := fetchPlaylists(ctx, ds, "owner_name")

	if outputFormat == "csv" {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"playlist id", "playlist name", "owner id", "owner name", "public"})
		for _, playlist := range allPls {
			_ = w.Write([]string{playlist.ID, playlist.Name, playlist.OwnerID, playlist.OwnerName, strconv.FormatBool(playlist.Public)})
		}
		w.Flush()
	} else {
		display := make(displayPlaylists, len(allPls))
		for idx, playlist := range allPls {
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

func runImport(ctx context.Context, files []string) {
	ds, ctx := getAdminContext(ctx)

	if userID != "" {
		user, err := getUser(ctx, userID, ds)
		if err != nil {
			log.Fatal(ctx, "Error retrieving user", "username or id", userID)
		}
		ctx = request.WithUser(ctx, *user)
	}

	pls := playlists.NewPlaylists(ds, core.NewImageUploadService())

	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			log.Error("Error resolving path", "file", file, err)
			fmt.Fprintf(os.Stderr, "Error: could not resolve path %s: %v\n", file, err)
			continue
		}

		totalLines := countM3UTrackLines(absPath)

		imported, err := pls.ImportFile(ctx, absPath, syncFlag)
		if err != nil {
			log.Error("Error importing playlist", "file", absPath, err)
			fmt.Fprintf(os.Stderr, "Error importing %s: %v\n", file, err)
			continue
		}

		matched := len(imported.Tracks)
		if totalLines > 0 {
			notFound := totalLines - matched
			fmt.Printf("Imported \"%s\" — %d/%d tracks matched (%d not found)\n", imported.Name, matched, totalLines, notFound)
		} else {
			fmt.Printf("Imported \"%s\" — %d tracks\n", imported.Name, matched)
		}
	}
}

func countM3UTrackLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	reader := ioutils.UTF8Reader(file)
	for line := range slice.LinesFrom(reader) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		count++
	}
	return count
}
