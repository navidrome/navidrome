package podcasts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// DownloadEpisode marks the episode downloaded for the current user (from ctx) - see the
// Podcasts interface doc for why this doubles as "resolve instantly from an existing shared file"
// for a second subscriber. The actual fetch (if needed) and the episode Put() both run against an
// admin context, since writing shared infrastructure state must succeed regardless of whether the
// requesting subscriber happens to be an admin.
func (p *podcasts) DownloadEpisode(ctx context.Context, episodeID string) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return errors.New("downloading an episode requires an authenticated user")
	}
	episode, err := p.ds.PodcastEpisode(ctx).Get(episodeID)
	if err != nil {
		return err
	}
	if err := p.ds.PodcastEpisode(ctx).SetDownloaded(user.ID, true, episodeID); err != nil {
		return fmt.Errorf("marking episode downloaded for user: %w", err)
	}
	if episode.IsDownloaded() {
		return nil
	}
	adminCtx := adminContext(ctx)
	p.downloadOne(adminCtx, episode)
	if episode.DownloadStatus != model.PodcastEpisodeDownloaded {
		return fmt.Errorf("downloading episode %s: %s", episodeID, episode.ErrorMessage)
	}
	return nil
}

// DeleteEpisode clears the current user's (from ctx) own downloaded flag. The shared file is
// only actually removed once no subscriber anywhere still has the episode flagged - see
// cleanupOrphanedFile in retention.go, called here for just this one episode rather than a full
// retention sweep, so unflagging feels immediate rather than waiting for the next scheduled run.
func (p *podcasts) DeleteEpisode(ctx context.Context, id string) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return errors.New("deleting an episode requires an authenticated user")
	}
	if err := p.ds.PodcastEpisode(ctx).SetDownloaded(user.ID, false, id); err != nil {
		return err
	}
	return p.cleanupOrphanedFile(adminContext(ctx), id)
}

// cleanupOrphanedFile deletes the shared file for episodeID only if no user, anywhere, still has
// it flagged as downloaded - called after any single user's flag is cleared (DeleteEpisode,
// per-subscriber retention eviction), so the file lingers as long as even one subscriber still
// wants it, and disappears promptly once the last one gives it up rather than waiting for the
// next scheduled retention sweep. ctx must carry admin/system privileges.
func (p *podcasts) cleanupOrphanedFile(ctx context.Context, episodeID string) error {
	stillWanted, err := p.ds.PodcastEpisode(ctx).AnyUserWantsDownload(episodeID)
	if err != nil {
		return fmt.Errorf("checking remaining demand for episode %s: %w", episodeID, err)
	}
	if stillWanted {
		return nil
	}
	episode, err := p.ds.PodcastEpisode(ctx).Get(episodeID)
	if err != nil {
		return err
	}
	if !episode.IsDownloaded() {
		return nil
	}
	return p.deleteEpisodeFile(ctx, episode)
}

// downloadForSubscriber applies sub's download policy to episodes - policy None does nothing,
// New/All both mark every episode in the list as downloaded for sub's user and fetch the shared
// file if it doesn't already exist. Callers control which episodes this means in practice:
// RefreshChannel passes only newly-discovered episodes (so New and All behave identically for a
// fresh episode), while Subscribe's initial backfill only calls this at all when policy is All,
// passing the channel's full existing episode list.
func (p *podcasts) downloadForSubscriber(ctx context.Context, channel model.PodcastChannel, sub model.PodcastSubscription, episodes model.PodcastEpisodes) {
	if sub.DownloadPolicy == model.PodcastDownloadPolicyNone || len(episodes) == 0 {
		return
	}
	ids := make([]string, len(episodes))
	for i, ep := range episodes {
		ids[i] = ep.ID
	}
	if err := p.ds.PodcastEpisode(ctx).SetDownloaded(sub.UserID, true, ids...); err != nil {
		log.Error(ctx, "Error marking episodes downloaded for subscriber", "channelId", channel.ID, "userId", sub.UserID, err)
		return
	}
	adminCtx := adminContext(ctx)
	var toFetch []model.PodcastEpisode
	for _, ep := range episodes {
		if !ep.IsDownloaded() {
			toFetch = append(toFetch, ep)
		}
	}
	p.downloadEpisodes(adminCtx, toFetch)
}

// deleteEpisodeFile removes a downloaded episode's local file (if any) and
// marks it deleted, shared by both the on-demand DeleteEpisode API and
// scheduled retention cleanup. Also removes the episode from any playlist
// it was added to - a playlist can only ever reference a downloaded
// episode, so once the download is gone the playlist entry would otherwise
// be left dangling.
func (p *podcasts) deleteEpisodeFile(ctx context.Context, episode *model.PodcastEpisode) error {
	if episode.Path != "" {
		if err := os.Remove(episodeAbsolutePath(episode.Path)); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warn(ctx, "Error removing downloaded podcast episode file", "id", episode.ID, "path", episode.Path, err)
		}
	}
	episode.Path = ""
	episode.Suffix = ""
	episode.Size = 0
	episode.DownloadStatus = model.PodcastEpisodeDeleted
	episode.ErrorMessage = ""
	if err := p.ds.PodcastEpisode(ctx).Put(episode); err != nil {
		return err
	}
	if err := p.ds.Playlist(ctx).RemoveItemFromPlaylists(episode.ID); err != nil {
		log.Warn(ctx, "Error removing deleted podcast episode from playlists", "id", episode.ID, err)
	}
	p.notifyDownload(ctx, episode)
	return nil
}

// downloadEpisodes concurrently downloads the given episodes to disk,
// bounded by Server.Podcasts.DownloadConcurrency, mirroring the go-pipeline
// pattern the library scanner uses for concurrent work. Per-episode
// failures are logged and recorded on the episode; they don't abort the
// batch.
func (p *podcasts) downloadEpisodes(ctx context.Context, episodes []model.PodcastEpisode) {
	if len(episodes) == 0 {
		return
	}

	concurrency := conf.Server.Podcasts.DownloadConcurrency
	if concurrency == 0 {
		concurrency = 1
	}

	producer := ppl.NewProducer(func(put func(model.PodcastEpisode)) error {
		for _, ep := range episodes {
			put(ep)
		}
		return nil
	}, ppl.Name("enqueue podcast downloads"))

	stage := ppl.NewStage(func(ep model.PodcastEpisode) (model.PodcastEpisode, error) {
		p.downloadOne(ctx, &ep)
		return ep, nil
	}, ppl.Name("download podcast episode"), ppl.Concurrency(concurrency))

	if err := ppl.Do(producer, stage); err != nil {
		log.Error(ctx, "Error running podcast download pipeline", err)
	}
}

func (p *podcasts) downloadOne(ctx context.Context, ep *model.PodcastEpisode) {
	p.setDownloadStatus(ctx, ep, model.PodcastEpisodeDownloading, "")

	relPath, suffix, size, err := fetchEpisodeToDisk(ctx, *ep)
	if err != nil {
		log.Error(ctx, "Error downloading podcast episode", "id", ep.ID, "title", ep.Title, "url", ep.EnclosureUrl, err)
		p.setDownloadStatus(ctx, ep, model.PodcastEpisodeDownloadError, err.Error())
		return
	}

	ep.Path = relPath
	ep.Suffix = suffix
	ep.Size = size
	ep.DownloadStatus = model.PodcastEpisodeDownloaded
	ep.ErrorMessage = ""
	if err := p.ds.PodcastEpisode(ctx).Put(ep); err != nil {
		log.Error(ctx, "Error saving downloaded podcast episode", "id", ep.ID, err)
		return
	}
	p.notifyDownload(ctx, ep)
}

func (p *podcasts) setDownloadStatus(ctx context.Context, ep *model.PodcastEpisode, status model.PodcastEpisodeDownloadStatus, errMsg string) {
	ep.DownloadStatus = status
	ep.ErrorMessage = errMsg
	if err := p.ds.PodcastEpisode(ctx).Put(ep, "DownloadStatus", "ErrorMessage"); err != nil {
		log.Error(ctx, "Error updating podcast episode download status", "id", ep.ID, err)
	}
	p.notifyDownload(ctx, ep)
}

// fetchEpisodeToDisk downloads an episode's enclosure to a temp file, then
// atomically renames it into place, returning the storage-relative path,
// suffix and actual byte size (not the often-wrong advertised enclosure
// length).
func fetchEpisodeToDisk(ctx context.Context, ep model.PodcastEpisode) (relPath, suffix string, size int64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.EnclosureUrl, nil)
	if err != nil {
		return "", "", 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetching episode: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("unexpected status %d downloading episode", resp.StatusCode)
	}

	suffix = suffixFor(resp.Header.Get("Content-Type"), ep.EnclosureUrl)
	relPath = episodeStoragePath(ep, suffix)
	absPath := episodeAbsolutePath(relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("creating podcast storage directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(absPath), ".download-*")
	if err != nil {
		return "", "", 0, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	written, err := io.Copy(tmp, resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("writing episode to disk: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", "", 0, fmt.Errorf("closing downloaded episode file: %w", err)
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		return "", "", 0, fmt.Errorf("finalizing downloaded episode file: %w", err)
	}

	return relPath, suffix, written, nil
}
