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
)

func (p *podcasts) DownloadEpisode(ctx context.Context, episodeID string) error {
	episode, err := p.ds.PodcastEpisode(ctx).Get(episodeID)
	if err != nil {
		return err
	}
	if episode.IsDownloaded() {
		return nil
	}
	p.downloadOne(ctx, episode)
	if episode.DownloadStatus != model.PodcastEpisodeDownloaded {
		return fmt.Errorf("downloading episode %s: %s", episodeID, episode.ErrorMessage)
	}
	return nil
}

func (p *podcasts) DeleteEpisode(ctx context.Context, id string) error {
	episode, err := p.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}
	return p.deleteEpisodeFile(ctx, episode)
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
