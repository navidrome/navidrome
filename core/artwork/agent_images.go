package artwork

import (
	"context"
	"io"
	"net/url"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

// externalName mirrors the normalization the aggregate provider applies, so agent searches match.
func externalName(name string) string {
	if conf.Server.DevPreserveUnicodeInExternalCalls {
		return name
	}
	return str.Clear(name)
}

// bestImageURL returns the largest fetchable image URL. Only one is returned and its failure ends
// the agent's turn, so an unfetchable candidate must never win: url.Parse alone accepts anything.
func bestImageURL(imgs []agents.ExternalImage) *url.URL {
	var best *url.URL
	var bestSize int
	for i := range imgs {
		if imgs[i].URL == "" {
			continue
		}
		u, err := url.Parse(imgs[i].URL)
		if err != nil || !u.IsAbs() || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		if best == nil || imgs[i].Size > bestSize {
			best, bestSize = u, imgs[i].Size
		}
	}
	return best
}

// fetchArtistImage tries each enabled artist-image agent in order. extErr is true only when no
// agent succeeded and at least one failed transiently.
func fetchArtistImage(ctx context.Context, ag *agents.Agents, gate gateFunc, ar model.Artist) (r io.ReadCloser, agentName string, extErr bool) {
	// Synthetic artists would otherwise get an unrelated agent result assigned to them.
	switch ar.ID {
	case consts.UnknownArtistID, consts.VariousArtistsID:
		traceFrom(ctx).add(TraceStep{Candidate: externalCandidate, Outcome: OutcomeSkipped, Detail: "synthetic artist"})
		return nil, "", false
	}
	name := externalName(ar.Name)
	imageAgents := ag.ArtistImageAgents()
	if len(imageAgents) == 0 {
		traceFrom(ctx).add(TraceStep{Candidate: externalCandidate, Outcome: OutcomeSkipped,
			Detail: "no enabled agent provides artist images"})
		return nil, "", false
	}
	for _, a := range imageAgents {
		reader, _, err := gate(a.Name, func() (io.ReadCloser, string, error) {
			imgs, err := a.Retriever.GetArtistImages(ctx, ar.ID, name, ar.MbzArtistID)
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
			log.Debug(ctx, "Artwork: External artist-image lookup failed", "agent", a.Name, "artist", ar.Name, err)
		}
	}
	return nil, "", extErr
}

// fetchAlbumImage is the album counterpart of fetchArtistImage.
func fetchAlbumImage(ctx context.Context, ag *agents.Agents, gate gateFunc, al model.Album) (r io.ReadCloser, agentName string, extErr bool) {
	name, artist := externalName(al.Name), externalName(al.AlbumArtist)
	imageAgents := ag.AlbumImageAgents()
	if len(imageAgents) == 0 {
		traceFrom(ctx).add(TraceStep{Candidate: externalCandidate, Outcome: OutcomeSkipped,
			Detail: "no enabled agent provides album images"})
		return nil, "", false
	}
	for _, a := range imageAgents {
		reader, _, err := gate(a.Name, func() (io.ReadCloser, string, error) {
			imgs, err := a.Retriever.GetAlbumImages(ctx, name, artist, al.MbzAlbumID)
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
			log.Debug(ctx, "Artwork: External album-image lookup failed", "agent", a.Name, "album", al.Name, err)
		}
	}
	return nil, "", extErr
}
