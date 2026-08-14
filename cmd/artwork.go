package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/spf13/cobra"
)

var artworkKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork,
	model.KindPlaylistArtwork, model.KindRadioArtwork,
}

var explainLive bool

var (
	reprocessKinds   []string
	reprocessSources []string
	reprocessAll     bool
	reprocessDryRun  bool
	reprocessYes     bool
)

func init() {
	artworkExplainCmd.Flags().BoolVar(&explainLive, "live", false,
		"perform real external lookups instead of reporting what would be tried")
	artworkReprocessCmd.Flags().StringSliceVar(&reprocessKinds, "kind", nil,
		"kinds to reprocess (ar, al, pl, ra); repeatable")
	artworkReprocessCmd.Flags().StringSliceVar(&reprocessSources, "source", nil,
		"only items currently resolved from these sources (e.g. folder, external:deezer, absent)")
	artworkReprocessCmd.Flags().BoolVar(&reprocessAll, "all", false, "reprocess every kind")
	artworkReprocessCmd.Flags().BoolVar(&reprocessDryRun, "dry-run", false,
		"report what would be queued and exit without queueing")
	artworkReprocessCmd.Flags().BoolVarP(&reprocessYes, "yes", "y", false, "skip the confirmation prompt")
	artworkCmd.AddCommand(artworkExplainCmd)
	artworkCmd.AddCommand(artworkRefreshCmd)
	artworkCmd.AddCommand(artworkReprocessCmd)
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

var artworkRefreshCmd = &cobra.Command{
	Use:   "refresh <kind> <id>...",
	Short: "Clear an item's artwork state and re-resolve it",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		kind, err := parseArtworkKind(args[0])
		if err != nil {
			log.Fatal(cmd.Context(), err)
		}
		runRefresh(cmd.Context(), kind, args[1:])
	},
}

var artworkReprocessCmd = &cobra.Command{
	Use:   "reprocess",
	Short: "Re-enqueue artwork in bulk, by kind and/or by the source it currently resolves from",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runReprocess(cmd.Context())
	},
}

func runReprocess(ctx context.Context) {
	kinds, err := selectedKinds(reprocessKinds, reprocessSelectsAll(reprocessKinds, reprocessSources, reprocessAll))
	if err != nil {
		log.Fatal(ctx, err)
	}

	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	if err := reprocessArtwork(ctx, ds, kinds, repositorySources(reprocessSources),
		reprocessDryRun, reprocessConfirm(reprocessYes, os.Stdin), os.Stdout); err != nil {
		log.Fatal(ctx, err)
	}
}

// reprocessSelectsAll reports whether every kind is targeted: a source filter on its own is already
// a complete selection, so it does not also need a kind.
func reprocessSelectsAll(kinds, sources []string, all bool) bool {
	return all || (len(kinds) == 0 && len(sources) > 0)
}

func selectedKinds(kinds []string, all bool) ([]model.Kind, error) {
	if all {
		return artworkKinds, nil
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("no selector given: pass --kind, --source or --all")
	}
	out := make([]model.Kind, 0, len(kinds))
	for _, k := range kinds {
		kind, err := parseArtworkKind(k)
		if err != nil {
			return nil, err
		}
		// A repeated kind would be counted twice, overstating the cost the operator confirms.
		if !slices.Contains(out, kind) {
			out = append(out, kind)
		}
	}
	return out, nil
}

// absentSource is how the stored empty source — resolved, no image — is spelled on the CLI.
const absentSource = "absent"

func repositorySources(sources []string) []string {
	return slice.Map(sources, func(s string) string {
		if s == absentSource {
			return ""
		}
		return s
	})
}

func displaySource(s string) string { return cmp.Or(s, absentSource) }

// confirmFunc reports whether the operator accepted queueing total items, of which external may
// reach an external agent.
type confirmFunc func(out io.Writer, total, external int64) bool

// reprocessConfirm is the only bypass of the confirmation: --yes, for scripted use.
func reprocessConfirm(yes bool, in io.Reader) confirmFunc {
	if yes {
		return func(io.Writer, int64, int64) bool { return true }
	}
	return promptConfirm(in)
}

func promptConfirm(in io.Reader) confirmFunc {
	return func(out io.Writer, total, external int64) bool {
		cost := fmt.Sprintf(", requiring up to %d external lookups", external)
		if external == 0 {
			cost = ""
		}
		fmt.Fprintf(out, "\nThis will re-resolve %d items%s. Continue? [y/N] ", total, cost)
		var answer string
		if _, err := fmt.Fscanln(in, &answer); err != nil {
			return false
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		return answer == "y" || answer == "yes"
	}
}

// validateSources rejects a typo'd source: matching nothing silently reads as "nothing to do" when
// it means the filter was wrong. Checked table-wide, so a filter is never a typo for one --kind only.
func validateSources(q model.ArtworkQueueRepository, sources []string) error {
	if len(sources) == 0 {
		return nil
	}
	var inUse []string
	for _, k := range artworkKinds {
		found, err := q.SourcesInUse(k)
		if err != nil {
			return fmt.Errorf("listing the sources in use by %s artwork: %w", k, err)
		}
		for _, s := range found {
			if !slices.Contains(inUse, s) {
				inUse = append(inUse, s)
			}
		}
	}
	var unknown []string
	for _, s := range sources {
		if !slices.Contains(inUse, s) {
			unknown = append(unknown, displaySource(s))
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	valid := slice.Map(inUse, displaySource)
	slices.Sort(valid)
	return fmt.Errorf("no artwork resolves from %s; sources in use: %s",
		strings.Join(unknown, ", "), cmp.Or(strings.Join(valid, ", "), "(none)"))
}

// reprocessArtwork previews from CountBySource — rows matched — then reports what EnqueueBySource
// actually inserted; the two differ because an already-queued row is left untouched.
func reprocessArtwork(ctx context.Context, ds model.DataStore, kinds []model.Kind, sources []string,
	dryRun bool, confirm confirmFunc, out io.Writer) error {
	q := ds.ArtworkQueue(ctx)
	if err := validateSources(q, sources); err != nil {
		return err
	}

	matched := make([]int64, len(kinds))
	var total, external int64
	for i, k := range kinds {
		n, err := q.CountBySource(k, sources)
		if err != nil {
			return fmt.Errorf("counting %s artwork: %w", k, err)
		}
		matched[i] = n
		total += n
		if walksPriorityChain(k) {
			external += n
		}
	}
	printReprocessPreview(out, kinds, matched, total, external, sources)

	if total == 0 {
		fmt.Fprintln(out, "\nNothing matches this selection.")
	}
	switch {
	case dryRun:
		fmt.Fprintln(out, "\nDry run: nothing was queued.")
		return nil
	case total == 0:
		fmt.Fprintln(out, "Nothing was queued.")
		return nil
	case !confirm(out, total, external):
		fmt.Fprintln(out, "Aborted: nothing was queued.")
		return nil
	}

	var queued int64
	for i, k := range kinds {
		if matched[i] == 0 {
			continue
		}
		n, err := q.EnqueueBySource(k, sources, model.ArtworkPriorityRecheck)
		if err != nil {
			return fmt.Errorf("queueing %s artwork: %w", k, err)
		}
		queued += n
		fmt.Fprintf(out, "%s: %d queued\n", k, n)
	}
	fmt.Fprintf(out, "Queued %d of %d matched items.\n", queued, total)
	if skipped := total - queued; skipped > 0 {
		fmt.Fprintf(out, "Already queued, left unchanged: %d (priority and retry backoff untouched).\n", skipped)
	}
	return nil
}

// printReprocessPreview also states the external estimate, which --dry-run must show precisely
// because it skips the prompt that would otherwise carry it.
func printReprocessPreview(out io.Writer, kinds []model.Kind, matched []int64, total, external int64, sources []string) {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	shown := slice.Map(sources, displaySource)
	fmt.Fprintf(w, "Sources:\t%s\n\n", cmp.Or(strings.Join(shown, ", "), "(any)"))
	fmt.Fprintln(w, "KIND\tMATCHED")
	for i, k := range kinds {
		fmt.Fprintf(w, "%s\t%d\n", k, matched[i])
	}
	fmt.Fprintf(w, "TOTAL\t%d\n", total)
	w.Flush()

	estimate := fmt.Sprintf("up to %d", external)
	if external == 0 {
		estimate = "none"
	}
	fmt.Fprintf(out, "\nExternal lookups: %s\n", estimate)
}

func runRefresh(ctx context.Context, kind model.Kind, ids []string) {
	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	if failed := refreshItems(ctx, ds, kind, ids, os.Stdout); failed > 0 {
		log.Fatal(ctx, "Failed to refresh artwork", "kind", kind, "failed", failed, "total", len(ids))
	}
}

// refreshItems keeps going after a failure — the ids are independent — and returns how many failed.
func refreshItems(ctx context.Context, ds model.DataStore, kind model.Kind, ids []string, out io.Writer) int {
	var failed int
	for _, id := range ids {
		// artwork.Refresh would happily queue an id that does not exist, orphaning a queue row.
		if _, err := artworkItemName(ctx, ds, kind, id); err != nil {
			log.Error(ctx, "Item not found", "kind", kind, "id", id, err)
			failed++
			continue
		}
		if err := artwork.Refresh(ctx, ds, kind, id); err != nil {
			log.Error(ctx, "Error refreshing artwork", "kind", kind, "id", id, err)
			failed++
			continue
		}
		fmt.Fprintf(out, "%s/%s: queued\n", kind.Prefix(), id)
	}
	return failed
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
