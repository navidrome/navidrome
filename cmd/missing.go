package cmd

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
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
	Title     string `json:"title"`
	Album     string `json:"album"`
	Artist    string `json:"artist"`
	Path      string `json:"path"`
}

func runMissingList(ctx context.Context) {
	if missingListFormat != "csv" && missingListFormat != "json" {
		log.Fatal("Invalid output format. Must be one of csv, json", "format", missingListFormat)
	}

	ds, ctx := getAdminContext(ctx)
	mfs, err := ds.MediaFile(ctx).GetCursor(model.QueryOptions{
		Filters: squirrel.Eq{"missing": true},
		Sort:    "path",
	})
	if err == nil {
		err = writeMissingList(os.Stdout, missingListFormat, mfs)
	}
	if err != nil {
		log.Fatal(ctx, "Failed to retrieve missing files", err)
	}
}

// writeMissingList streams the cursor so a library with many missing files doesn't get loaded into memory
func writeMissingList(w io.Writer, format string, mfs model.MediaFileCursor) error {
	if format == "json" {
		bw := bufio.NewWriter(w)
		_, _ = io.WriteString(bw, "[")
		sep := ""
		for mf, err := range mfs {
			if err != nil {
				return err
			}
			j, _ := json.Marshal(displayMissingFile{ID: mf.ID, LibraryID: mf.LibraryID, Title: mf.Title, Album: mf.Album, Artist: mf.Artist, Path: mf.Path})
			_, _ = fmt.Fprintf(bw, "%s%s", sep, j)
			sep = ","
		}
		_, _ = io.WriteString(bw, "]\n")
		return bw.Flush()
	}

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "library id", "title", "album", "artist", "path"})
	for mf, err := range mfs {
		if err != nil {
			return err
		}
		_ = cw.Write([]string{mf.ID, strconv.Itoa(mf.LibraryID), mf.Title, mf.Album, mf.Artist, mf.Path})
	}
	cw.Flush()
	return cw.Error()
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

// resolveMediaFile looks up a media file by ID first, then by path (optionally libraryID:path).
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
	mfs = preferQualified(ref, mfs)
	if len(mfs) > 1 {
		log.Fatal(ctx, "Path matches multiple files; disambiguate with an ID or libraryID:path", "ref", ref, "matches", len(mfs))
	}
	return &mfs[0]
}

// preferQualified resolves the ambiguity FindByPaths creates by searching a "libraryID:path"
// reference both ways: an explicit library wins over a file literally named like one.
func preferQualified(ref string, mfs model.MediaFiles) model.MediaFiles {
	id, path, ok := strings.Cut(ref, ":")
	if !ok {
		return mfs
	}
	libraryID, err := strconv.Atoi(id)
	if err != nil {
		return mfs
	}
	qualified := slice.Filter(mfs, func(mf model.MediaFile) bool {
		return mf.LibraryID == libraryID && strings.EqualFold(mf.Path, path)
	})
	if len(qualified) == 0 {
		return mfs
	}
	return qualified
}
