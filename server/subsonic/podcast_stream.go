package subsonic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const onDemandDownloadTimeout = 60 * time.Second

// streamPodcastEpisode is the fallback Stream() takes when id doesn't
// resolve to a model.MediaFile. It serves a downloaded episode's local
// file directly, proxies the source URL for stream-only channels (so a
// Subsonic client never needs to know about the external URL), or
// triggers an on-demand download for channels configured to download.
func (api *Router) streamPodcastEpisode(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) error {
	episode, err := api.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}

	// Mark listened as soon as a client starts streaming, mirroring the
	// "starting" state a song's ReportPlayback records - simpler than
	// tracking playback position/completion thresholds, which podcast
	// episodes don't support today.
	if err := api.ds.PodcastEpisode(ctx).IncPlayCount(episode.ID, time.Now()); err != nil {
		log.Warn(ctx, "Error recording podcast episode play", "id", id, err)
	}

	if episode.IsDownloaded() {
		if err := serveLocalPodcastFile(w, r, episode, ""); err == nil {
			return nil
		}
		log.Warn(ctx, "Downloaded podcast episode file missing, falling back to proxy", "id", id, "path", episode.Path)
	}

	channel, err := api.ds.PodcastChannel(ctx).Get(episode.ChannelID)
	if err != nil {
		return err
	}

	if channel.DownloadPolicy == model.PodcastDownloadPolicyNone {
		return proxyPodcastEnclosure(ctx, w, r, episode)
	}

	downloadCtx, cancel := context.WithTimeout(ctx, onDemandDownloadTimeout)
	defer cancel()
	if err := api.podcasts.DownloadEpisode(downloadCtx, id); err != nil {
		log.Warn(ctx, "Error downloading podcast episode on demand, falling back to proxy", "id", id, err)
		return proxyPodcastEnclosure(ctx, w, r, episode)
	}
	refreshed, err := api.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}
	if refreshed.IsDownloaded() {
		return serveLocalPodcastFile(w, r, refreshed, "")
	}
	return proxyPodcastEnclosure(ctx, w, r, episode)
}

// downloadPodcastEpisodeFile backs the Download() handler's
// *model.PodcastEpisode case: downloads the episode on demand if needed,
// then serves it as an attachment.
func (api *Router) downloadPodcastEpisodeFile(ctx context.Context, w http.ResponseWriter, r *http.Request, episode *model.PodcastEpisode) error {
	if !episode.IsDownloaded() {
		downloadCtx, cancel := context.WithTimeout(ctx, onDemandDownloadTimeout)
		defer cancel()
		if err := api.podcasts.DownloadEpisode(downloadCtx, episode.ID); err != nil {
			return err
		}
		refreshed, err := api.ds.PodcastEpisode(ctx).Get(episode.ID)
		if err != nil {
			return err
		}
		episode = refreshed
	}
	if !episode.IsDownloaded() {
		return model.ErrNotFound
	}
	disposition := fmt.Sprintf("attachment; filename=\"%s.%s\"", strings.ReplaceAll(episode.Title, ",", "_"), episode.Suffix)
	return serveLocalPodcastFile(w, r, episode, disposition)
}

func serveLocalPodcastFile(w http.ResponseWriter, r *http.Request, episode *model.PodcastEpisode, disposition string) error {
	f, err := os.Open(episode.AbsolutePath()) //nolint:gosec // path is built from server-generated channel/episode IDs (see naming.go), not user input
	if err != nil {
		return fmt.Errorf("opening downloaded podcast episode file: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "episode."+episode.Suffix, info.ModTime(), f)
	return nil
}

// proxyPodcastEnclosure streams a not-yet-downloaded episode's enclosure
// URL through this server, so stream-only channels are still playable by
// Subsonic clients (which only ever talk to /rest/stream.view). Does not
// currently forward Range requests, so seeking on a not-yet-downloaded
// episode may not work on every upstream host.
func proxyPodcastEnclosure(ctx context.Context, w http.ResponseWriter, r *http.Request, episode *model.PodcastEpisode) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, episode.EnclosureUrl, nil) //nolint:gosec // URL is the channel's RSS enclosure URL, set by the admin when subscribing to the feed
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // see above: URL is admin-configured, not arbitrary user input
	if err != nil {
		return fmt.Errorf("proxying podcast episode: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned status %d for podcast episode", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, resp.Body)
	return err
}
