package artwork

import (
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
)

// gateFunc gates one named external fetch (rate limit + circuit breaker per name).
// resolveItem defaults to passthroughGate; the worker injects the per-agent gate.
type gateFunc = func(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error)

func passthroughGate(_ string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return f()
}

// denyGate refuses every external fetch with a definitive not-found, so local-only
// resolution never runs a network step even if an external branch is reached.
func denyGate(_ string, _ func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return nil, "", model.ErrNotFound
}

// bestImageURL returns the largest-Size image URL, skipping empty or unparseable
// URLs; nil when none qualifies.
func bestImageURL(imgs []agents.ExternalImage) *url.URL {
	var best *agents.ExternalImage
	for i := range imgs {
		if imgs[i].URL == "" {
			continue
		}
		if best == nil || imgs[i].Size > best.Size {
			best = &imgs[i]
		}
	}
	if best == nil {
		return nil
	}
	u, err := url.Parse(best.URL)
	if err != nil {
		return nil
	}
	return u
}

// fetchArtistImage tries each enabled artist-image agent in order, each under its own gate.
// Returns the winning reader + agent name; extErr is true only when NO agent succeeded and
// at least one failed transiently (a later success beats an earlier agent error).
func fetchArtistImage(ctx context.Context, ag *agents.Agents, gate gateFunc, ar model.Artist) (r io.ReadCloser, agentName string, extErr bool) {
	for _, a := range ag.ArtistImageAgents() {
		reader, _, err := gate(a.Name, func() (io.ReadCloser, string, error) {
			imgs, err := a.Retriever.GetArtistImages(ctx, ar.ID, ar.Name, ar.MbzArtistID)
			if err != nil {
				return nil, "", err
			}
			u := bestImageURL(imgs)
			if u == nil {
				return nil, "", agents.ErrNotFound
			}
			return fromURL(ctx, u)
		})
		if reader != nil {
			return reader, a.Name, false
		}
		if isTransientExternal(err) {
			extErr = true // includes errBreakerOpen and download failures: retry via the next agent
		}
	}
	return nil, "", extErr
}

// fetchAlbumImage is the album counterpart of fetchArtistImage.
func fetchAlbumImage(ctx context.Context, ag *agents.Agents, gate gateFunc, al model.Album) (r io.ReadCloser, agentName string, extErr bool) {
	for _, a := range ag.AlbumImageAgents() {
		reader, _, err := gate(a.Name, func() (io.ReadCloser, string, error) {
			imgs, err := a.Retriever.GetAlbumImages(ctx, al.Name, al.AlbumArtist, al.MbzAlbumID)
			if err != nil {
				return nil, "", err
			}
			u := bestImageURL(imgs)
			if u == nil {
				return nil, "", agents.ErrNotFound
			}
			return fromURL(ctx, u)
		})
		if reader != nil {
			return reader, a.Name, false
		}
		if isTransientExternal(err) {
			extErr = true
		}
	}
	return nil, "", extErr
}

// isTransientExternal reports whether an external step failed in a way worth retrying;
// a not-found (from either package) is a definitive answer, not a fault.
func isTransientExternal(err error) bool {
	return err != nil && !errors.Is(err, agents.ErrNotFound) && !errors.Is(err, model.ErrNotFound)
}
