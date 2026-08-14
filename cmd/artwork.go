package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/spf13/cobra"
)

var artworkKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork,
	model.KindPlaylistArtwork, model.KindRadioArtwork,
}

var explainLive bool

func init() {
	artworkExplainCmd.Flags().BoolVar(&explainLive, "live", false,
		"perform real external lookups instead of reporting what would be tried")
	artworkCmd.AddCommand(artworkExplainCmd)
	rootCmd.AddCommand(artworkCmd)
}

var artworkCmd = &cobra.Command{
	Use:   "artwork",
	Short: "Inspect and re-resolve artwork",
}

var artworkExplainCmd = &cobra.Command{
	Use:   "explain <kind> <id>",
	Short: "Explain why an item's artwork resolved the way it did",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		kind, err := parseArtworkKind(args[0])
		if err != nil {
			log.Fatal(cmd.Context(), err)
		}
		runExplain(cmd.Context(), kind, args[1])
	},
}

func parseArtworkKind(s string) (model.Kind, error) {
	kind, ok := model.ParseKind(s)
	if ok {
		for _, k := range artworkKinds {
			if k == kind {
				return kind, nil
			}
		}
	}
	valid := make([]string, 0, len(artworkKinds))
	for _, k := range artworkKinds {
		valid = append(valid, k.Prefix())
	}
	return kind, fmt.Errorf("invalid kind %q, expected one of: %s", s, strings.Join(valid, ", "))
}

// walksPriorityChain reports whether the resolver picks this kind's artwork from a priority
// chain; playlists and radios resolve from a single source, so there is no walk to explain.
func walksPriorityChain(kind model.Kind) bool {
	return kind == model.KindArtistArtwork || kind == model.KindAlbumArtwork
}

// explainResult states the verdict of the walk. An external tier that was skipped or that failed
// leaves the outcome unknown: nothing observed that the item has no artwork.
func explainResult(source string, steps []artwork.TraceStep) string {
	if source != "" {
		for _, s := range steps {
			if s.Outcome == "hit" {
				break
			}
			if s.Outcome == "would-try" {
				return "resolved from " + source +
					" (offline: a higher-priority external candidate was not tried; re-run with --live)"
			}
		}
		return "resolved from " + source
	}
	for _, s := range steps {
		switch {
		case s.Outcome == "would-try":
			return "indeterminate (external agents not called; re-run with --live)"
		case s.Outcome == "error" && strings.HasPrefix(s.Candidate, "external:"):
			return "indeterminate (an external lookup failed; the item may resolve on a later attempt)"
		}
	}
	return "not resolved"
}

type explainReport struct {
	kind         model.Kind
	id           string
	name         string
	stored       *model.ItemArtwork
	queued       *model.ArtworkQueueItem
	priorityName string
	priority     string
	agents       string
	walksChain   bool
	steps        []artwork.TraceStep
	source       string
	resolveErr   error
}

func formatExplain(rep explainReport) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 4, 2, ' ', 0)

	fmt.Fprintln(w, "Item")
	fmt.Fprintf(w, "  Kind:\t%s (%s)\n", rep.kind, rep.kind.Prefix())
	fmt.Fprintf(w, "  ID:\t%s\n", rep.id)
	fmt.Fprintf(w, "  Name:\t%s\n", rep.name)

	fmt.Fprintln(w, "\nStored")
	if rep.stored == nil {
		fmt.Fprintln(w, "  (no artwork state recorded)")
	} else {
		fmt.Fprintf(w, "  Source:\t%s\n", cmp.Or(rep.stored.Source, "(absent)"))
		fmt.Fprintf(w, "  Hash:\t%s\n", cmp.Or(rep.stored.Hash, "(absent)"))
		if rep.stored.SourcePath != "" {
			fmt.Fprintf(w, "  Source path:\t%s\n", rep.stored.SourcePath)
		}
		fmt.Fprintf(w, "  Attempted at:\t%s\n", formatTime(rep.stored.AttemptedAt))
	}

	fmt.Fprintln(w, "\nQueue")
	if rep.queued == nil {
		fmt.Fprintln(w, "  (not queued)")
	} else {
		fmt.Fprintf(w, "  Priority:\t%d\n", rep.queued.Priority)
		fmt.Fprintf(w, "  Attempts:\t%d\n", rep.queued.Attempts)
		fmt.Fprintf(w, "  Retry at:\t%s\n", formatTime(rep.queued.RetryAt))
	}

	fmt.Fprintln(w, "\nConfig")
	if !rep.walksChain {
		fmt.Fprintln(w, "  (no priority chain configuration applies)")
	} else {
		fmt.Fprintf(w, "  %s:\t%s\n", rep.priorityName, rep.priority)
		fmt.Fprintf(w, "  Agents:\t%s\n", rep.agents)
	}

	fmt.Fprintln(w, "\nChain")
	if !rep.walksChain {
		fmt.Fprintf(w, "  (%s artwork does not walk a priority chain)\n", rep.kind)
	} else {
		fmt.Fprintln(w, "  CANDIDATE\tOUTCOME\tDETAIL")
		for _, s := range rep.steps {
			// A row with an empty last cell would end tabwriter's column block, breaking alignment.
			fmt.Fprintf(w, "  %s\t%s\t%s\n", s.Candidate, s.Outcome, cmp.Or(s.Detail, "-"))
		}
	}

	fmt.Fprintln(w, "\nResult")
	switch {
	case rep.resolveErr != nil:
		fmt.Fprintf(w, "  resolution failed: %s\n", rep.resolveErr)
	case !rep.walksChain:
		fmt.Fprintln(w, "  not evaluated (no chain was walked; see Stored above)")
	default:
		fmt.Fprintf(w, "  %s\n", explainResult(rep.source, rep.steps))
	}

	w.Flush()
	return sb.String()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func runExplain(ctx context.Context, kind model.Kind, id string) {
	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	name, err := artworkItemName(ctx, ds, kind, id)
	if err != nil {
		log.Fatal(ctx, "Item not found", "kind", kind, "id", id, err)
	}
	rep := explainReport{
		kind: kind, id: id, name: name,
		priorityName: "CoverArtPriority", priority: conf.Server.CoverArtPriority,
		agents:     conf.Server.Agents,
		walksChain: walksPriorityChain(kind),
	}
	if kind == model.KindArtistArtwork {
		rep.priorityName, rep.priority = "ArtistArtPriority", conf.Server.ArtistArtPriority
	}

	rep.stored, err = ds.Artwork(ctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Fatal(ctx, "Failed to read artwork state", "kind", kind, "id", id, err)
	}
	rep.queued, err = ds.ArtworkQueue(ctx).Get(kind, id, model.ImageTypePrimary)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Fatal(ctx, "Failed to read the artwork queue", "kind", kind, "id", id, err)
	}

	if rep.walksChain {
		trace := &artwork.ChainTrace{}
		resolver := CreateArtworkResolver(trace, explainLive)
		resolve := resolver.ResolveAlbum
		if kind == model.KindArtistArtwork {
			resolve = resolver.ResolveArtist
		}
		rep.source, rep.resolveErr = resolve(ctx, id)
		rep.steps = trace.Steps()
	}

	fmt.Print(formatExplain(rep))
	// The steps taken before a failed walk are the diagnosis, so report them before exiting.
	if rep.resolveErr != nil {
		log.Fatal(ctx, "Failed to resolve artwork", "kind", kind, "id", id, rep.resolveErr)
	}
}

// artworkItemName looks the entity up under its own kind, so a mismatched kind/id pair is
// reported as not found instead of silently explaining another entity's artwork.
func artworkItemName(ctx context.Context, ds model.DataStore, kind model.Kind, id string) (string, error) {
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
	}
	return "", fmt.Errorf("unsupported kind %q", kind.Prefix())
}
