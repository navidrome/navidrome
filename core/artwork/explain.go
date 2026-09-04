package artwork

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

// Result states the verdict of the walk. A skipped or failed external tier, or a local candidate
// that would not open, leaves the outcome unknown: nothing observed that there is no artwork.
func Result(source string, steps []TraceStep) string {
	if source != "" {
		for _, s := range steps {
			if s.Outcome == OutcomeHit {
				break
			}
			// An external winner discards the earlier error, so the resolver settles it with no retry.
			if s.Outcome == OutcomeError && strings.HasPrefix(s.Candidate, ExternalPrefix) &&
				!strings.HasPrefix(source, ExternalPrefix) {
				return "resolved from " + source +
					" (indeterminate: a higher-priority external lookup failed; this may resolve differently on a retry)"
			}
		}
		return "resolved from " + source
	}
	for _, s := range steps {
		switch {
		case s.Outcome == OutcomeError && strings.HasPrefix(s.Candidate, ExternalPrefix):
			return "indeterminate (an external lookup failed; the item may resolve on a later attempt)"
		case s.Outcome == OutcomeError, s.Outcome == OutcomeUnreadable:
			return "indeterminate (a candidate was found but could not be processed; the worker retries rather than settling absent)"
		}
	}
	return "not resolved"
}

// ConfigFor names the setting that decides where a kind's artwork comes from, and its value.
func ConfigFor(kind model.Kind) (setting, value string) {
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

// ImageAgentNames names the agents that can actually supply an image for kind.
func ImageAgentNames(ag *agents.Agents, kind model.Kind) []string {
	if kind == model.KindArtistArtwork {
		return slice.Map(ag.ArtistImageAgents(), func(a agents.ArtistImageAgent) string { return a.Name })
	}
	return slice.Map(ag.AlbumImageAgents(), func(a agents.AlbumImageAgent) string { return a.Name })
}

// FormatAgents accounts for every configured agent: one that cannot be constructed never reaches
// the chain, so the raw list alone overstates it. unavailableNote is appended only if some are.
func FormatAgents(configured string, available []string, unavailableNote string) string {
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
		line += unavailableNote
	}
	return line
}

// ExplainOptions configures a single explain. The zero value reads history and never
// touches the network.
type ExplainOptions struct {
	// Walk builds a resolver that records into the trace; nil reads the recorded trace instead.
	Walk func(*ChainTrace) *TracingResolver
	// UnavailableNote is appended to the agent line when a configured agent is missing.
	UnavailableNote string
}

// ExplainReport is everything known about how one item's artwork resolved.
type ExplainReport struct {
	Kind       model.Kind
	ID         string
	Name       string
	Stored     *model.ItemArtwork
	Queued     *model.ArtworkQueueItem
	Steps      []TraceStep
	Source     string
	Agents     string
	Walked     bool
	ResolveErr error
}

// Explain gathers everything known about how kind/id's artwork resolved: stored state, the
// queue row, and either the recorded trace or a fresh walk, depending on opts.Walk.
func Explain(ctx context.Context, ds model.DataStore, ag *agents.Agents, kind model.Kind, id string,
	opts ExplainOptions) (ExplainReport, error) {
	name, err := ItemName(ctx, ds, kind, id)
	if err != nil {
		return ExplainReport{}, err
	}
	rep := ExplainReport{Kind: kind, ID: id, Name: name}

	if KeepsState(kind) {
		rep.Stored, err = ds.Artwork(ctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return ExplainReport{}, fmt.Errorf("reading artwork state: %w", err)
		}
		rep.Queued, err = ds.ArtworkQueue(ctx).Get(kind, id, model.ImageTypePrimary)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return ExplainReport{}, fmt.Errorf("reading the artwork queue: %w", err)
		}
	}
	if !Explainable(kind) {
		return rep, nil
	}
	if ag != nil && (kind == model.KindArtistArtwork || kind == model.KindAlbumArtwork) {
		rep.Agents = FormatAgents(conf.Server.Agents, ImageAgentNames(ag, kind), opts.UnavailableNote)
	}

	rep.Walked = opts.Walk != nil
	switch {
	case rep.Walked:
		trace := &ChainTrace{}
		rep.Source, rep.ResolveErr = opts.Walk(trace).Resolve(ctx, kind, id)
		rep.Steps = trace.Steps()
	case rep.Stored != nil:
		rep.Steps = DecodeTrace(rep.Stored.Trace, rep.Stored.SourcePath)
		rep.Source = rep.Stored.Source
	}
	return rep, nil
}

// Result reports this report's verdict; see the package-level Result for the rules.
func (r ExplainReport) Result() string { return Result(r.Source, r.Steps) }

// ChainOrigin says whether the report reads history or a walk performed just now, since the two
// can disagree after a config change.
func (r ExplainReport) ChainOrigin() string {
	if r.Walked {
		return "walked now"
	}
	if r.Stored != nil {
		return "recorded " + r.Stored.AttemptedAt.Format(time.RFC3339)
	}
	return "not recorded"
}

// LastAttemptFailed decodes why the queued row's last attempt failed, if there is one queued.
func (r ExplainReport) LastAttemptFailed() []TraceStep {
	if r.Queued == nil {
		return nil
	}
	return DecodeTrace(r.Queued.Trace, "")
}

// GaveUpAfter decodes the trace of the attempt that exhausted the retry budget, if the stored
// state recorded one.
func (r ExplainReport) GaveUpAfter() []TraceStep {
	if r.Stored == nil {
		return nil
	}
	return DecodeTrace(r.Stored.LastFailure, "")
}
