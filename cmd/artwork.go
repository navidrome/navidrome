package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/spf13/cobra"
)

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
	artworkCmd.AddCommand(artworkStatusCmd)
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

var artworkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report the artwork queue, where artwork resolves from, and the backfill state",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runStatus(cmd.Context())
	},
}

func runStatus(ctx context.Context) {
	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	rep, err := collectStatus(ctx, ds)
	if err != nil {
		log.Fatal(ctx, err)
	}
	fmt.Print(formatStatus(rep))
}

type sourceCount struct {
	kind   model.Kind
	source string
	count  int64
}

type absentCount struct {
	kind model.Kind
	model.ArtworkAbsentStat
}

type statusReport struct {
	queue   []model.ArtworkQueueStat
	sources []sourceCount
	absent  []absentCount
	inputs  []artwork.FingerprintInput
	stored  string
	current string
}

func (r statusReport) queueTotal() int64 {
	var n int64
	for _, s := range r.queue {
		n += s.Count
	}
	return n
}

func (r statusReport) backfillQueued() int64 {
	var n int64
	for _, s := range r.queue {
		if s.Priority == model.ArtworkPriorityBackfill {
			n += s.Count
		}
	}
	return n
}

func collectStatus(ctx context.Context, ds model.DataStore) (statusReport, error) {
	q := ds.ArtworkQueue(ctx)
	var rep statusReport
	var err error
	if rep.queue, err = q.CountByKindAndPriority(); err != nil {
		return rep, fmt.Errorf("breaking the artwork queue down by kind: %w", err)
	}

	cutoff := time.Now().Add(-artwork.StaleAbsentAge)
	for _, k := range artwork.RecheckKinds {
		sources, err := q.SourcesInUse(k)
		if err != nil {
			return rep, fmt.Errorf("listing the sources in use by %s artwork: %w", k, err)
		}
		slices.Sort(sources)
		for _, s := range sources {
			n, err := q.CountBySource(k, []string{s})
			if err != nil {
				return rep, fmt.Errorf("counting %s artwork resolved from %s: %w", k, displaySource(s), err)
			}
			rep.sources = append(rep.sources, sourceCount{kind: k, source: s, count: n})
		}
		stat, err := q.CountAbsent(k, cutoff)
		if err != nil {
			return rep, fmt.Errorf("counting absent %s artwork: %w", k, err)
		}
		rep.absent = append(rep.absent, absentCount{kind: k, ArtworkAbsentStat: stat})
	}

	rep.current, rep.inputs = artwork.ConfigFingerprint(), artwork.FingerprintInputs()
	if rep.stored, err = ds.Property(ctx).DefaultGet(consts.ArtConfFingerprintPropertyKey, ""); err != nil {
		return rep, fmt.Errorf("reading the stored artwork fingerprint: %w", err)
	}
	return rep, nil
}

func formatStatus(rep statusReport) string {
	var sb strings.Builder
	w := newTabWriter(&sb)

	fmt.Fprintln(w, "Queue")
	if len(rep.queue) == 0 {
		fmt.Fprintln(w, "  (empty)")
	} else {
		fmt.Fprintln(w, "  KIND\tPRIORITY\tITEMS")
		for _, s := range rep.queue {
			fmt.Fprintf(w, "  %s\t%s\t%d\n", kindName(s.ItemKind), priorityName(s.Priority), s.Count)
		}
		fmt.Fprintf(w, "  TOTAL\t\t%d\n", rep.queueTotal())
	}

	fmt.Fprintln(w, "\nSources")
	fmt.Fprintln(w, "  KIND\tSOURCE\tITEMS")
	for _, s := range rep.sources {
		fmt.Fprintf(w, "  %s\t%s\t%d\n", s.kind, displaySource(s.source), s.count)
	}

	fmt.Fprintln(w, "\nAbsent (resolved, no image found)")
	fmt.Fprintln(w, "  KIND\tABSENT\tDUE FOR RECHECK")
	for _, a := range rep.absent {
		fmt.Fprintf(w, "  %s\t%d\t%d\n", a.kind, a.Total, a.Stale)
	}
	fmt.Fprintf(w, "  (rechecked once the last attempt is older than %gh)\n", artwork.StaleAbsentAge.Hours())

	fmt.Fprintln(w, "\nBackfill")
	fmt.Fprintf(w, "  State:\t%s\n", backfillState(rep))
	fmt.Fprintf(w, "  Stored fingerprint:\t%s\n", cmp.Or(rep.stored, "(none)"))
	fmt.Fprintf(w, "  Current fingerprint:\t%s\n", rep.current)
	if len(rep.inputs) > 0 {
		fmt.Fprintln(w, "  Fingerprint inputs (changing any of these re-resolves the whole library):")
		for _, in := range rep.inputs {
			fmt.Fprintf(w, "    %s:\t%s\n", in.Name, in.Value)
		}
	}

	w.Flush()
	return sb.String()
}

// backfillState leads with the queued backlog: by the time anyone runs this, backfill has usually
// already stored the new fingerprint, and "up to date" would bury the flood it is still working through.
func backfillState(rep statusReport) string {
	pending := "fingerprint changed — every artist, album, playlist and radio will be re-enqueued on the next startup"
	if n := rep.backfillQueued(); n > 0 {
		if rep.stored != rep.current {
			return fmt.Sprintf("backfill running: %d items queued, and %s", n, pending)
		}
		return fmt.Sprintf("backfill running: %d items queued (fingerprint up to date)", n)
	}
	if rep.stored != rep.current {
		return pending
	}
	return "up to date"
}

func kindName(prefix string) string {
	if k, ok := model.ParseKind(prefix); ok {
		return k.String()
	}
	return prefix
}

func priorityName(p int) string {
	switch p {
	case model.ArtworkPriorityRecheck:
		return "recheck"
	case model.ArtworkPriorityBackfill:
		return "backfill"
	case model.ArtworkPriorityScan:
		return "scan"
	case model.ArtworkPriorityBump:
		return "bump"
	}
	return strconv.Itoa(p)
}

func runReprocess(ctx context.Context) {
	kinds, err := selectedKinds(reprocessKinds, reprocessSources, reprocessAll)
	if err != nil {
		log.Fatal(ctx, err)
	}

	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	if err := reprocessArtwork(ctx, ds, kinds, repositorySources(reprocessSources), imageAgentCount(ds),
		reprocessDryRun, reprocessConfirm(reprocessYes, os.Stdin), os.Stdout); err != nil {
		log.Fatal(ctx, err)
	}
}

func selectedKinds(kinds, sources []string, all bool) ([]model.Kind, error) {
	// A source filter on its own is already a complete selection, so it does not also need a kind.
	if all || (len(kinds) == 0 && len(sources) > 0) {
		return artwork.RecheckKinds, nil
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
		out = append(out, kind)
	}
	// A repeated kind would be counted twice, overstating the cost the operator confirms.
	return slice.Unique(out), nil
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

type confirmFunc func(out io.Writer, total, external int64) bool

func reprocessConfirm(yes bool, in io.Reader) confirmFunc {
	if yes {
		return func(io.Writer, int64, int64) bool { return true }
	}
	return promptConfirm(in)
}

// externalEstimate claims no bound: a local hit ends the walk before any agent is asked, and plugin
// agents are unregistered in a CLI that never starts the plugin manager.
func externalEstimate(n int64) string {
	if n == 0 {
		return "none"
	}
	return fmt.Sprintf("~%d estimated (plugin agents not counted; local hits may need fewer)", n)
}

func externalLookupLine(n int64) string {
	return fmt.Sprintf("External lookups: %s.", externalEstimate(n))
}

// imageAgentCount counts only the built-in image agents, for the same reason.
func imageAgentCount(ds model.DataStore) artwork.ImageAgentCount {
	ag := agents.GetAgents(ds, getPluginManager())
	return artwork.ImageAgentCount{Artist: len(ag.ArtistImageAgents()), Album: len(ag.AlbumImageAgents())}
}

func promptConfirm(in io.Reader) confirmFunc {
	return func(out io.Writer, total, external int64) bool {
		var cost string
		if external > 0 {
			cost = fmt.Sprintf(" %s", externalLookupLine(external))
		}
		fmt.Fprintf(out, "\nThis will re-resolve %d items.%s Continue? [y/N] ", total, cost)
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
	for _, k := range artwork.RecheckKinds {
		found, err := q.SourcesInUse(k)
		if err != nil {
			return fmt.Errorf("listing the sources in use by %s artwork: %w", k, err)
		}
		inUse = slice.Unique(append(inUse, found...))
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
	imageAgents artwork.ImageAgentCount, dryRun bool, confirm confirmFunc, out io.Writer) error {
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
		external += n * artwork.ExternalLookupsPerItem(k, imageAgents)
	}
	printReprocessPreview(out, kinds, matched, total, external, sources)

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

// printReprocessPreview also states the external estimate, which --dry-run must show because it
// skips the prompt that would otherwise carry it.
func printReprocessPreview(out io.Writer, kinds []model.Kind, matched []int64, total, external int64, sources []string) {
	w := newTabWriter(out)
	shown := slice.Map(sources, displaySource)
	fmt.Fprintf(w, "Sources:\t%s\n\n", cmp.Or(strings.Join(shown, ", "), "(any)"))
	fmt.Fprintln(w, "KIND\tMATCHED")
	for i, k := range kinds {
		fmt.Fprintf(w, "%s\t%d\n", k, matched[i])
	}
	fmt.Fprintf(w, "TOTAL\t%d\n", total)
	w.Flush()

	fmt.Fprintf(out, "\n%s\n", externalLookupLine(external))
	if total == 0 {
		fmt.Fprintln(out, "\nNothing matches this selection.")
	}
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
	if ok && slices.Contains(artwork.RecheckKinds, kind) {
		return kind, nil
	}
	valid := slice.Map(artwork.RecheckKinds, func(k model.Kind) string { return k.Prefix() })
	return kind, fmt.Errorf("invalid kind %q, expected one of: %s", s, strings.Join(valid, ", "))
}

// explainAgents accounts for every configured agent: one the CLI cannot construct (a plugin, or a
// built-in missing its credentials) never reaches the Chain, so the raw list alone overstates it.
func explainAgents(configured string, available []string) string {
	if strings.TrimSpace(configured) == "" {
		return "(none)"
	}
	var unavailable bool
	names := slice.Map(strings.Split(configured, ","), func(name string) string {
		name = strings.TrimSpace(name)
		if slices.Contains(available, name) {
			return name
		}
		unavailable = true
		return name + "*"
	})
	line := strings.Join(names, ", ")
	if unavailable {
		line += "  (* not available to the CLI)"
	}
	return line
}

// availableImageAgents names the agents that can actually supply an image for kind.
func availableImageAgents(ds model.DataStore, kind model.Kind) []string {
	ag := agents.GetAgents(ds, getPluginManager())
	if kind == model.KindArtistArtwork {
		return slice.Map(ag.ArtistImageAgents(), func(a agents.ArtistImageAgent) string { return a.Name })
	}
	return slice.Map(ag.AlbumImageAgents(), func(a agents.AlbumImageAgent) string { return a.Name })
}

// explainResult states the verdict of the walk. An external tier that was skipped or that failed
// leaves the outcome unknown: nothing observed that the item has no artwork.
func explainResult(source string, steps []artwork.TraceStep) string {
	if source != "" {
		for _, s := range steps {
			if s.Outcome == artwork.OutcomeHit {
				break
			}
			if s.Outcome == artwork.OutcomeWouldTry {
				return "resolved from " + source +
					" (offline: a higher-priority external candidate was not tried; re-run with --live)"
			}
			// An external winner discards the earlier error, so the resolver settles it with no retry.
			if s.Outcome == artwork.OutcomeError && strings.HasPrefix(s.Candidate, artwork.ExternalPrefix) &&
				!strings.HasPrefix(source, artwork.ExternalPrefix) {
				return "resolved from " + source +
					" (indeterminate: a higher-priority external lookup failed; this may resolve differently on a retry)"
			}
		}
		return "resolved from " + source
	}
	for _, s := range steps {
		switch {
		case s.Outcome == artwork.OutcomeWouldTry:
			return "indeterminate (external agents not called; re-run with --live)"
		case s.Outcome == artwork.OutcomeError && strings.HasPrefix(s.Candidate, artwork.ExternalPrefix):
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
	steps        []artwork.TraceStep
	source       string
	resolveErr   error
}

func formatExplain(rep explainReport) string {
	var sb strings.Builder
	w := newTabWriter(&sb)
	walksChain := artwork.WalksPriorityChain(rep.kind)

	fmt.Fprintln(w, "Item")
	fmt.Fprintf(w, "  Kind:\t%s (%s)\n", rep.kind, rep.kind.Prefix())
	fmt.Fprintf(w, "  ID:\t%s\n", rep.id)
	fmt.Fprintf(w, "  Name:\t%s\n", rep.name)

	fmt.Fprintln(w, "\nStored")
	if rep.stored == nil {
		fmt.Fprintln(w, "  (no artwork state recorded)")
	} else {
		fmt.Fprintf(w, "  Source:\t%s\n", displaySource(rep.stored.Source))
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
		fmt.Fprintf(w, "  Priority:\t%s (%d)\n", priorityName(rep.queued.Priority), rep.queued.Priority)
		fmt.Fprintf(w, "  Attempts:\t%d\n", rep.queued.Attempts)
		fmt.Fprintf(w, "  Retry at:\t%s\n", formatTime(rep.queued.RetryAt))
	}

	fmt.Fprintln(w, "\nConfig")
	if !walksChain {
		fmt.Fprintln(w, "  (no priority chain configuration applies)")
	} else {
		fmt.Fprintf(w, "  %s:\t%s\n", rep.priorityName, rep.priority)
		fmt.Fprintf(w, "  Agents:\t%s\n", rep.agents)
	}

	fmt.Fprintln(w, "\nChain")
	if !walksChain {
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
	case !walksChain:
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

	if artwork.WalksPriorityChain(kind) {
		rep.agents = explainAgents(conf.Server.Agents, availableImageAgents(ds, kind))
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
