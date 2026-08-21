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
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/spf13/cobra"
)

var missingListFormat string

func init() {
	missingListCmd.Flags().StringVarP(&missingListFormat, "format", "f", "csv", "output format [supported values: csv, json]")
	missingCmd.AddCommand(missingListCmd)
	missingCmd.AddCommand(missingFixCmd)
	rootCmd.AddCommand(missingCmd)
}

var (
	missingCmd = &cobra.Command{
		Use:   "missing",
		Short: "Manage missing files",
		Long:  "List files marked as missing and remap them onto existing files",
	}

	missingListCmd = &cobra.Command{
		Use:   "list",
		Short: "List missing files",
		Run: func(cmd *cobra.Command, _ []string) {
			runMissingList(cmd.Context())
		},
	}

	missingFixCmd = &cobra.Command{
		Use:   "fix <missing path|id> <target path|id>",
		Short: "Remap a missing file onto an existing file",
		Long: "Remap a file marked as missing onto an existing (non-missing) file, the same way\n" +
			"the scanner reconciles moved or renamed files. Each argument may be a media file ID,\n" +
			"a library-relative path, or a libraryID:path pair.",
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			runMissingFix(cmd.Context(), args[0], args[1])
		},
	}
)

type displayMissingFile struct {
	ID        string `json:"id"`
	LibraryID int    `json:"libraryId"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Album     string `json:"album"`
	Artist    string `json:"artist"`
}

func runMissingList(ctx context.Context) {
	if missingListFormat != "csv" && missingListFormat != "json" {
		log.Fatal("Invalid output format. Must be one of csv, json", "format", missingListFormat)
	}

	ds, ctx := getAdminContext(ctx)
	mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"missing": true},
		Sort:    "path",
	})
	if err != nil {
		log.Fatal(ctx, "Failed to retrieve missing files", err)
	}

	if missingListFormat == "json" {
		display := make([]displayMissingFile, len(mfs))
		for i, mf := range mfs {
			display[i] = displayMissingFile{ID: mf.ID, LibraryID: mf.LibraryID, Path: mf.Path, Title: mf.Title, Album: mf.Album, Artist: mf.Artist}
		}
		j, _ := json.Marshal(display)
		fmt.Printf("%s\n", j)
	} else {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"id", "library id", "path", "title", "album", "artist"})
		for _, mf := range mfs {
			_ = w.Write([]string{mf.ID, strconv.Itoa(mf.LibraryID), mf.Path, mf.Title, mf.Album, mf.Artist})
		}
		w.Flush()
	}
}

func runMissingFix(ctx context.Context, missingRef, targetRef string) {
	ds, ctx := getAdminContext(ctx)

	missing := resolveMediaFile(ctx, ds, missingRef)
	target := resolveMediaFile(ctx, ds, targetRef)

	if err := core.NewMaintenance(ds).RemapMissingFile(ctx, missing.ID, target.ID); err != nil {
		log.Fatal(ctx, "Failed to remap missing file", "missing", missing.Path, "target", target.Path, err)
	}
	fmt.Printf("Remapped %q onto %q\n", missing.Path, target.Path)
}

// resolveMediaFile looks up a media file by ID first, then by path (optionally libraryID:path),
// following the same "try one, then the other" pattern used by findPlaylist.
func resolveMediaFile(ctx context.Context, ds model.DataStore, ref string) *model.MediaFile {
	mf, err := ds.MediaFile(ctx).Get(ref)
	if err == nil {
		return mf
	}
	if !errors.Is(err, model.ErrNotFound) {
		log.Fatal(ctx, "Error looking up media file", "ref", ref, err)
	}

	mfs, err := ds.MediaFile(ctx).FindByPaths([]string{ref})
	if err != nil {
		log.Fatal(ctx, "Error looking up media file by path", "ref", ref, err)
	}
	if len(mfs) == 0 {
		log.Fatal(ctx, "No media file found", "ref", ref)
	}
	if len(mfs) > 1 {
		log.Fatal(ctx, "Path matches multiple files; disambiguate with an ID or libraryID:path", "ref", ref, "matches", len(mfs))
	}
	return &mfs[0]
}
