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
	"github.com/navidrome/navidrome/plugins"
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

var (
	cancelKinds      []string
	cancelPriorities []string
	cancelAll        bool
	cancelDryRun     bool
	cancelYes        bool
)

func init() {
	artworkExplainCmd.Flags().BoolVar(&explainLive, "live", false,
		"walk the chain again now, performing real external lookups, instead of reporting the "+
			"stored trace of the last resolution; also initializes plugin agents, which may open "+
			"external connections")
	artworkReprocessCmd.Flags().StringSliceVar(&reprocessKinds, "kind", nil,
		"kinds to reprocess ("+kindPrefixes(artwork.RecheckKinds)+"); repeatable")
	artworkReprocessCmd.Flags().StringSliceVar(&reprocessSources, "source", nil,
		"only items currently resolved from these sources (e.g. folder, external:deezer, absent)")
	artworkReprocessCmd.Flags().BoolVar(&reprocessAll, "all", false, "reprocess every kind")
	artworkReprocessCmd.Flags().BoolVar(&reprocessDryRun, "dry-run", false,
		"report what would be queued and exit without queueing")
	artworkReprocessCmd.Flags().BoolVarP(&reprocessYes, "yes", "y", false, "skip the confirmation prompt")
	artworkCancelCmd.Flags().StringSliceVar(&cancelKinds, "kind", nil,
		"kinds to cancel ("+kindPrefixes(artwork.RefreshableKinds)+"); repeatable")
	artworkCancelCmd.Flags().StringSliceVar(&cancelPriorities, "priority", nil,
		"only rows queued at these priorities ("+priorityNames()+"); repeatable")
	artworkCancelCmd.Flags().BoolVar(&cancelAll, "all", false, "cancel every kind at every priority")
	artworkCancelCmd.Flags().BoolVar(&cancelDryRun, "dry-run", false,
		"report what would be cancelled and exit without cancelling")
	artworkCancelCmd.Flags().BoolVarP(&cancelYes, "yes", "y", false, "skip the confirmation prompt")
	artworkCmd.AddCommand(artworkExplainCmd)
	artworkCmd.AddCommand(artworkRefreshCmd)
	artworkCmd.AddCommand(artworkReprocessCmd)
	artworkCmd.AddCommand(artworkCancelCmd)
	artworkCmd.AddCommand(artworkStatusCmd)
	rootCmd.AddCommand(artworkCmd)
}

var artworkCmd = &cobra.Command{
	Use:   "artwork",
	Short: "Inspect and re-resolve artwork",
}

var artworkExplainCmd = &cobra.Command{
	Use:   "explain [<kind>] <id>",
	Short: "Explain why an item's artwork resolved the way it did",
	Long: "Explain why an item's artwork resolved the way it did.\n\n" +
		"The item can be given as a bare id, a full artwork id (e.g. al-<id>), or a <kind> <id> pair.\n" +
		"<kind> is one of: " + kindPrefixes(explainKinds) + ".\n" +
		"A disc artwork id is the album id and the disc number, joined by a colon: <albumID>:2",
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		runExplain(cmd.Context(), args)
	},
}

var artworkRefreshCmd = &cobra.Command{
	Use:   "refresh [<kind>] <id>...",
	Short: "Clear an item's artwork state and re-resolve it",
	Long: "Clear an item's artwork state and re-resolve it.\n\n" +
		"Each item can be given as a bare id, a full artwork id (e.g. al-<id>), or a shared\n" +
		"<kind> <id>... leader. <kind> is one of: " + kindPrefixes(artwork.RefreshableKinds) + ".",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runRefresh(cmd.Context(), args)
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

var artworkCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel pending artwork work in bulk, by kind and/or queue priority",
	Long: "Cancel pending artwork work in bulk, by kind and/or queue priority.\n\n" +
		"Only the queue is touched: resolved artwork and the state behind `artwork explain` are\n" +
		"left alone, and the trace of why a cancelled item last failed goes with its queue row.\n\n" +
		"Work already picked up is not interrupted, and an item with no artwork yet can be\n" +
		"queued again by the hourly re-check. Use it to call off a bulk backfill, not to stop\n" +
		"the worker.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runCancel(cmd.Context())
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

func (r statusReport) queueTotal() int64 { return queueTotal(r.queue) }

func queueTotal(stats []model.ArtworkQueueStat) int64 {
	var n int64
	for _, s := range stats {
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
	if rep.queue, err = q.CountQueued(nil, nil); err != nil {
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
		printQueueStats(w, rep.queue, rep.queueTotal(), "ITEMS", "  ")
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

// printQueueStats writes the shared queue breakdown; the caller owns the tab writer and flushes it.
func printQueueStats(w io.Writer, stats []model.ArtworkQueueStat, total int64, countHeader, indent string) {
	fmt.Fprintf(w, "%sKIND\tPRIORITY\t%s\n", indent, countHeader)
	for _, s := range stats {
		fmt.Fprintf(w, "%s%s\t%s\t%d\n", indent, kindName(s.ItemKind), priorityName(s.Priority), s.Count)
	}
	fmt.Fprintf(w, "%sTOTAL\t\t%d\n", indent, total)
}

func kindName(prefix string) string {
	if k, ok := model.ParseKind(prefix); ok {
		return k.String()
	}
	return prefix
}

type artworkPriority struct {
	name  string
	value int
}

// artworkPriorities is the one listing behind both the name and the parse, so they cannot drift.
var artworkPriorities = []artworkPriority{
	{"bump", model.ArtworkPriorityBump},
	{"scan", model.ArtworkPriorityScan},
	{"backfill", model.ArtworkPriorityBackfill},
	{"recheck", model.ArtworkPriorityRecheck},
}

// priorityName falls back to the number: a row written by a newer version still has to print.
func priorityName(p int) string {
	for _, ap := range artworkPriorities {
		if ap.value == p {
			return ap.name
		}
	}
	return strconv.Itoa(p)
}

func priorityNames() string {
	return strings.Join(slice.Map(artworkPriorities, func(ap artworkPriority) string { return ap.name }), ", ")
}

func parseArtworkPriority(s string) (int, error) {
	for _, ap := range artworkPriorities {
		if ap.name == s {
			return ap.value, nil
		}
	}
	return 0, fmt.Errorf("invalid priority %q, expected one of: %s", s, priorityNames())
}

func runReprocess(ctx context.Context) {
	kinds, err := selectedKinds(reprocessKinds, reprocessSources, reprocessAll)
	if err != nil {
		log.Fatal(ctx, err)
	}

	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	// Only a kind that can reach an agent needs the count, and loading a plugin creates its
	// services. A preview must not reach the network, so init never runs here.
	var imageAgents artwork.ImageAgentCount
	if needsImageAgents(kinds) {
		mgr := loadPluginAgents(ctx, false)
		defer func() { _ = mgr.Stop() }()
		imageAgents = imageAgentCount(ds, mgr)
	}

	if err := reprocessArtwork(ctx, ds, kinds, repositorySources(reprocessSources), imageAgents,
		reprocessDryRun, confirmUnlessYes(reprocessYes, os.Stdin, "re-resolve"), os.Stdout); err != nil {
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
	return parseAll(kinds, func(s string) (model.Kind, error) {
		return parseArtworkKind(s, artwork.RecheckKinds)
	})
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

func confirmUnlessYes(yes bool, in io.Reader, verb string) confirmFunc {
	if yes {
		return func(io.Writer, int64, int64) bool { return true }
	}
	return promptConfirm(in, verb)
}

// externalEstimate claims no bound: a local hit ends the walk before any agent is asked, and the
// plugin agents it counts are only the ones this process managed to load.
func externalEstimate(n int64) string {
	if n == 0 {
		return "none"
	}
	return fmt.Sprintf("~%d estimated (plugin agents counted only when they load; local hits may need fewer)", n)
}

func externalLookupLine(n int64) string {
	return fmt.Sprintf("External lookups: %s.", externalEstimate(n))
}

func imageAgentCount(ds model.DataStore, mgr *plugins.Manager) artwork.ImageAgentCount {
	ag := agents.GetAgents(ds, mgr)
	return artwork.ImageAgentCount{Artist: len(ag.ArtistImageAgents()), Album: len(ag.AlbumImageAgents())}
}

// loadPluginAgents loads the plugins named in Agents, so the CLI resolves through the same agents a
// running server would. A load failure is reported, not fatal: the built-in agents still answer.
func loadPluginAgents(ctx context.Context, runInit bool) *plugins.Manager {
	mgr := getPluginManager()
	if err := mgr.LoadPlugins(ctx, configuredAgents(), runInit); err != nil {
		log.Warn(ctx, "Could not load plugins; plugin-provided agents will be missing", err)
	}
	return mgr
}

// needsImageAgents asks exactly what ExternalLookupsPerItem asks, so the gate cannot disagree with
// the estimate it guards. Playlists count: their generated grid resolves album art through agents.
func needsImageAgents(kinds []model.Kind) bool {
	return slices.ContainsFunc(kinds, artwork.MayFetchExternal)
}

// configuredAgents names the agents in priority order; one absent from it can never supply an image.
func configuredAgents() []string {
	var names []string
	for name := range strings.SplitSeq(conf.Server.Agents, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func promptConfirm(in io.Reader, verb string) confirmFunc {
	return func(out io.Writer, total, external int64) bool {
		var cost string
		if external > 0 {
			cost = fmt.Sprintf(" %s", externalLookupLine(external))
		}
		fmt.Fprintf(out, "\nThis will %s %d items.%s Continue? [y/N] ", verb, total, cost)
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
		if s != "" && !slices.Contains(inUse, s) { // the reserved absent source is valid even when nothing is absent
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

func runCancel(ctx context.Context) {
	kinds, priorities, err := cancelSelection(cancelKinds, cancelPriorities, cancelAll)
	if err != nil {
		log.Fatal(ctx, err)
	}

	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	if err := cancelArtwork(ctx, ds, kinds, priorities, cancelDryRun,
		confirmUnlessYes(cancelYes, os.Stdin, "cancel"), os.Stdout); err != nil {
		log.Fatal(ctx, err)
	}
}

// cancelSelection leaves --all as the empty filter the repository reads as "every one", so a row
// whose kind this build does not know still gets cancelled.
func cancelSelection(kinds, priorities []string, all bool) ([]model.Kind, []int, error) {
	if all {
		return nil, nil, nil
	}
	if len(kinds) == 0 && len(priorities) == 0 {
		return nil, nil, fmt.Errorf("no selector given: pass --kind, --priority or --all")
	}
	// RefreshableKinds, not RecheckKinds: media files are queued, so --kind must reach them.
	outKinds, err := parseAll(kinds, func(s string) (model.Kind, error) {
		return parseArtworkKind(s, artwork.RefreshableKinds)
	})
	if err != nil {
		return nil, nil, err
	}
	outPriorities, err := parseAll(priorities, parseArtworkPriority)
	if err != nil {
		return nil, nil, err
	}
	return outKinds, outPriorities, nil
}

// parseAll drops repeats: a doubled selector would overstate the total the operator confirms.
func parseAll[T comparable](values []string, parse func(string) (T, error)) ([]T, error) {
	out := make([]T, 0, len(values))
	for _, v := range values {
		parsed, err := parse(v)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return slice.Unique(out), nil
}

func cancelArtwork(ctx context.Context, ds model.DataStore, kinds []model.Kind, priorities []int,
	dryRun bool, confirm confirmFunc, out io.Writer) error {
	q := ds.ArtworkQueue(ctx)
	matched, err := q.CountQueued(kinds, priorities)
	if err != nil {
		return fmt.Errorf("counting queued artwork: %w", err)
	}
	total := queueTotal(matched)
	w := newTabWriter(out)
	printQueueStats(w, matched, total, "MATCHED", "")
	w.Flush()

	switch {
	case total == 0:
		fmt.Fprintln(out, "\nNothing matches this selection.")
		return nil
	case dryRun:
		fmt.Fprintln(out, "\nDry run: nothing was cancelled.")
		return nil
	case !confirm(out, total, 0):
		fmt.Fprintln(out, "Aborted: nothing was cancelled.")
		return nil
	}

	cancelled, err := q.PurgeQueued(kinds, priorities)
	if err != nil {
		return fmt.Errorf("cancelling queued artwork: %w", err)
	}
	// Count and delete are separate statements, so a drain in between makes these two differ.
	fmt.Fprintf(out, "Cancelled %d of %d matched items.\n", cancelled, total)
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

func runRefresh(ctx context.Context, args []string) {
	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	targets, failures, err := resolveArtworkTargets(ctx, ds, args, artwork.RefreshableKinds)
	if err != nil {
		log.Fatal(ctx, err)
	}
	for _, f := range failures {
		log.Error(ctx, "Skipping unresolved item", f)
	}
	failed := refreshItems(ctx, ds, targets, os.Stdout) + len(failures)
	if failed > 0 {
		log.Fatal(ctx, "Failed to refresh artwork", "failed", failed, "total", len(targets)+len(failures))
	}
}

// refreshItems keeps going after a failure — the items are independent — and returns how many failed.
func refreshItems(ctx context.Context, ds model.DataStore, targets []model.ArtworkID, out io.Writer) int {
	var failed int
	for _, t := range targets {
		kind, id := t.Kind, t.ID
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

// explainKinds is every kind explain accepts: it reports stored state and config too, so a kind
// with no chain to walk still has something to answer with.
var explainKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindDiscArtwork,
	model.KindMediaFileArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

func kindPrefixes(kinds []model.Kind) string {
	return strings.Join(model.KindPrefixes(kinds), ", ")
}

func parseArtworkKind(s string, valid []model.Kind) (model.Kind, error) {
	kind, ok := model.ParseKind(s)
	if ok && slices.Contains(valid, kind) {
		return kind, nil
	}
	return kind, invalidKindErr(s, valid)
}

func invalidKindErr(s string, valid []model.Kind) error {
	return fmt.Errorf("invalid kind %q, expected one of: %s", s, kindPrefixes(valid))
}

// resolveArtworkTargets resolves explain/refresh positional args into artwork ids, accepting a
// shared "<kind> <id>..." leader or self-describing args (a bare id, or a full artwork id). A
// self-describing arg that cannot be resolved is returned as a failure rather than aborting the
// batch, so refresh can process the resolvable ids; a malformed <kind> leader is a usage error.
func resolveArtworkTargets(ctx context.Context, ds model.DataStore, args []string, valid []model.Kind) ([]model.ArtworkID, []error, error) {
	if kind, ok := model.ParseKind(args[0]); ok && len(args) > 1 {
		if !slices.Contains(valid, kind) {
			return nil, nil, invalidKindErr(args[0], valid)
		}
		return slice.Map(args[1:], func(id string) model.ArtworkID {
			return model.ArtworkID{Kind: kind, ID: id}
		}), nil, nil
	}
	var targets []model.ArtworkID
	var failures []error
	for _, arg := range args {
		target, err := artworkKindAndID(ctx, ds, arg)
		if err == nil && !slices.Contains(valid, target.Kind) {
			err = invalidKindErr(target.Kind.Prefix(), valid)
		}
		if err != nil {
			failures = append(failures, err)
			continue
		}
		targets = append(targets, target)
	}
	return targets, failures, nil
}

// artworkKindAndID resolves one self-describing argument: a full artwork id (al-<id>) takes its kind
// from the prefix, a bare id is looked up. Entity ids never start with "<kind>-", so no collision.
func artworkKindAndID(ctx context.Context, ds model.DataStore, arg string) (model.ArtworkID, error) {
	if artID, err := model.ParseArtworkID(arg); err == nil && artID.ID != "" {
		return model.ArtworkID{Kind: artID.Kind, ID: artID.ID}, nil
	}
	kind, err := model.GetEntityKindByID(ctx, ds, arg)
	if errors.Is(err, model.ErrNotFound) {
		return model.ArtworkID{}, fmt.Errorf("could not determine kind for %q; pass an explicit <kind>", arg)
	}
	if err != nil {
		return model.ArtworkID{}, err
	}
	return model.ArtworkID{Kind: kind, ID: arg}, nil
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
func availableImageAgents(ds model.DataStore, mgr *plugins.Manager, kind model.Kind) []string {
	ag := agents.GetAgents(ds, mgr)
	if kind == model.KindArtistArtwork {
		return slice.Map(ag.ArtistImageAgents(), func(a agents.ArtistImageAgent) string { return a.Name })
	}
	return slice.Map(ag.AlbumImageAgents(), func(a agents.AlbumImageAgent) string { return a.Name })
}

// explainResult states the verdict of the walk. A skipped or failed external tier, or a local
// candidate that would not open, leaves the outcome unknown: nothing observed that there is no artwork.
func explainResult(source string, steps []artwork.TraceStep) string {
	if source != "" {
		for _, s := range steps {
			if s.Outcome == artwork.OutcomeHit {
				break
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
		case s.Outcome == artwork.OutcomeError && strings.HasPrefix(s.Candidate, artwork.ExternalPrefix):
			return "indeterminate (an external lookup failed; the item may resolve on a later attempt)"
		// A stage error or an unreadable candidate means a source was found but not processed; the
		// worker retries rather than settling absent, so neither reads as a clean miss.
		case s.Outcome == artwork.OutcomeError, s.Outcome == artwork.OutcomeUnreadable:
			return "indeterminate (a candidate was found but could not be processed; the worker retries rather than settling absent)"
		}
	}
	return "not resolved"
}

// explainConfig names the setting that decides where a kind's artwork comes from, and its value.
func explainConfig(kind model.Kind) (name, value string) {
	switch kind {
	case model.KindArtistArtwork:
		return "ArtistArtPriority", conf.Server.ArtistArtPriority
	case model.KindAlbumArtwork:
		return "CoverArtPriority", conf.Server.CoverArtPriority
	case model.KindDiscArtwork:
		return "DiscArtPriority", conf.Server.DiscArtPriority
	case model.KindMediaFileArtwork:
		return "EnableMediaFileCoverArt", strconv.FormatBool(conf.Server.EnableMediaFileCoverArt)
	}
	return "", ""
}

type explainReport struct {
	kind   model.Kind
	id     string
	name   string
	stored *model.ItemArtwork
	queued *model.ArtworkQueueItem
	agents string
	// steps is the chain walk: recorded when the item was resolved, or performed just now when walked.
	steps      []artwork.TraceStep
	source     string
	walked     bool
	resolveErr error
}

// explainChainOrigin says whether the operator is reading history or a walk performed just now,
// since the two can disagree after a config change.
func explainChainOrigin(rep explainReport) string {
	if rep.walked {
		return "walked now"
	}
	if rep.stored != nil {
		return "recorded " + formatTime(rep.stored.AttemptedAt)
	}
	return "not recorded"
}

// writeSteps prints the trace rows. An empty last cell would end tabwriter's column block and
// break the alignment, so a missing detail is rendered as a dash.
func writeSteps(w io.Writer, indent string, steps []artwork.TraceStep) {
	for _, s := range steps {
		fmt.Fprintf(w, "%s%s\t%s\t%s\n", indent, s.Candidate, s.Outcome, cmp.Or(s.Detail, "-"))
	}
}

// writeStepTable prints a secondary trace, and nothing at all when there is none to show.
func writeStepTable(w io.Writer, title string, steps []artwork.TraceStep) {
	if len(steps) == 0 {
		return
	}
	// No tab on the title: it closes the preceding column block, so these rows align among themselves.
	fmt.Fprintf(w, "  %s:\n", title)
	writeSteps(w, "    ", steps)
}

func formatExplain(rep explainReport) string {
	var sb strings.Builder
	w := newTabWriter(&sb)
	explainable := artwork.Explainable(rep.kind)
	stateful := artwork.KeepsState(rep.kind)
	unrecorded := !rep.walked && rep.stored == nil

	fmt.Fprintln(w, "Item")
	fmt.Fprintf(w, "  Kind:\t%s (%s)\n", rep.kind, rep.kind.Prefix())
	fmt.Fprintf(w, "  ID:\t%s\n", rep.id)
	fmt.Fprintf(w, "  Name:\t%s\n", rep.name)

	fmt.Fprintln(w, "\nStored")
	switch {
	case !stateful:
		fmt.Fprintf(w, "  (%s artwork is resolved on every request and never recorded)\n", rep.kind)
	case rep.stored == nil:
		fmt.Fprintln(w, "  (no artwork state recorded)")
	default:
		fmt.Fprintf(w, "  Source:\t%s\n", displaySource(rep.stored.Source))
		fmt.Fprintf(w, "  Hash:\t%s\n", cmp.Or(rep.stored.Hash, "(absent)"))
		if rep.stored.SourcePath != "" {
			fmt.Fprintf(w, "  Source path:\t%s\n", rep.stored.SourcePath)
		}
		fmt.Fprintf(w, "  Attempted at:\t%s\n", formatTime(rep.stored.AttemptedAt))
	}

	fmt.Fprintln(w, "\nQueue")
	switch {
	case !stateful:
		fmt.Fprintln(w, "  (never queued)")
	case rep.queued == nil:
		fmt.Fprintln(w, "  (not queued)")
	default:
		fmt.Fprintf(w, "  Priority:\t%s (%d)\n", priorityName(rep.queued.Priority), rep.queued.Priority)
		fmt.Fprintf(w, "  Attempts:\t%d\n", rep.queued.Attempts)
		fmt.Fprintf(w, "  Retry at:\t%s\n", formatTime(rep.queued.RetryAt))
	}
	if rep.queued != nil {
		writeStepTable(w, "Last attempt failed", artwork.DecodeTrace(rep.queued.Trace, ""))
	}
	if rep.stored != nil {
		writeStepTable(w, "Gave up after", artwork.DecodeTrace(rep.stored.LastFailure, ""))
	}

	fmt.Fprintln(w, "\nConfig")
	if setting, value := explainConfig(rep.kind); setting == "" {
		fmt.Fprintln(w, "  (no artwork source configuration applies)")
	} else {
		fmt.Fprintf(w, "  %s:\t%s\n", setting, value)
		if rep.agents != "" {
			fmt.Fprintf(w, "  Agents:\t%s\n", rep.agents)
		}
	}

	fmt.Fprintf(w, "\nChain (%s)\n", explainChainOrigin(rep))
	switch {
	case !explainable:
		fmt.Fprintf(w, "  (%s artwork does not walk a priority chain)\n", rep.kind)
	case unrecorded:
		fmt.Fprintln(w, "  (no resolution recorded yet; re-run with --live to walk the chain now)")
	case !rep.walked && len(rep.steps) == 0 && rep.stored.Hash != "":
		// A stored image with no chain can only predate trace recording: a recorded resolution that
		// found an image always records its winning candidate.
		fmt.Fprintln(w, "  (this item was resolved before traces were recorded; re-run with --live)")
	case !rep.walked && len(rep.steps) == 0:
		// Absent with no chain: an empty priority list walked nothing, or a pre-tracing absent row.
		fmt.Fprintln(w, "  (no candidates were recorded; re-run with --live to walk the chain now)")
	default:
		fmt.Fprintln(w, "  CANDIDATE\tOUTCOME\tDETAIL")
		writeSteps(w, "  ", rep.steps)
	}

	fmt.Fprintln(w, "\nResult")
	switch {
	case rep.resolveErr != nil:
		fmt.Fprintf(w, "  resolution failed: %s\n", rep.resolveErr)
	case !explainable:
		fmt.Fprintln(w, "  not evaluated (no chain was walked; see Stored above)")
	case unrecorded:
		fmt.Fprintln(w, "  not evaluated (nothing recorded; re-run with --live to walk the chain now)")
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

func runExplain(ctx context.Context, args []string) {
	defer db.Init(ctx)()
	ds, ctx := getAdminContext(ctx)

	targets, failures, err := resolveArtworkTargets(ctx, ds, args, explainKinds)
	if err != nil {
		log.Fatal(ctx, err)
	}
	if len(failures) > 0 {
		log.Fatal(ctx, failures[0])
	}
	if len(targets) != 1 {
		log.Fatal(ctx, "explain takes a single item; pass one id or a <kind> <id> pair")
	}
	kind, id := targets[0].Kind, targets[0].ID

	name, err := artworkItemName(ctx, ds, kind, id)
	if err != nil {
		log.Fatal(ctx, "Item not found", "kind", kind, "id", id, err)
	}
	rep := explainReport{kind: kind, id: id, name: name}
	if artwork.KeepsState(kind) {
		rep.stored, err = ds.Artwork(ctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			log.Fatal(ctx, "Failed to read artwork state", "kind", kind, "id", id, err)
		}
		rep.queued, err = ds.ArtworkQueue(ctx).Get(kind, id, model.ImageTypePrimary)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			log.Fatal(ctx, "Failed to read the artwork queue", "kind", kind, "id", id, err)
		}
	}

	// Disc artwork keeps no row, so it has no stored trace and can only be explained by walking now.
	rep.walked = explainLive || !artwork.KeepsState(kind)
	if artwork.Explainable(kind) {
		// Only artist and album reach an agent, and the load must precede the resolver, which reads
		// the same manager.
		if kind == model.KindArtistArtwork || kind == model.KindAlbumArtwork {
			mgr := loadPluginAgents(ctx, explainLive)
			defer func() { _ = mgr.Stop() }()
			rep.agents = explainAgents(conf.Server.Agents, availableImageAgents(ds, mgr, kind))
		}
		switch {
		case rep.walked:
			trace := &artwork.ChainTrace{}
			rep.source, rep.resolveErr = CreateArtworkResolver(trace, explainLive).Resolve(ctx, kind, id)
			rep.steps = trace.Steps()
		case rep.stored != nil:
			rep.steps = artwork.DecodeTrace(rep.stored.Trace, rep.stored.SourcePath)
			rep.source = rep.stored.Source
		}
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
