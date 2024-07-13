package core

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
)

var (
	userAgent = "Navidrome " + consts.Version
)

type deleteReq struct {
	id        string
	isPodcast bool
}

type podcastManager struct {
	// An empty string denotes a refresh, and a non-empty string
	// denotes a podcast to be downloaded
	ch     chan string
	client *http.Client
	del    chan deleteReq
	ds     model.DataStore
	parser *gofeed.Parser
}

type PodcastManager interface {
	CreateFeed(ctx context.Context, url string) (*model.Podcast, error)
	DeletePodcast(ctx context.Context, id string) error
	DeletePodcastEpisode(ctx context.Context, id string) error
	Download(ctx context.Context, id string) error
	Refresh(ctx context.Context) error
}

func NewPodcasts(ds model.DataStore) PodcastManager {
	parser := gofeed.NewParser()
	parser.UserAgent = userAgent

	client := http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}

	ch := make(chan string, 5)
	del := make(chan deleteReq, 5)

	podcasts := &podcastManager{
		ch:     ch,
		client: &client,
		del:    del,
		ds:     ds,
		parser: parser,
	}

	go podcasts.processDeletes()
	go podcasts.processFetch()

	return podcasts
}

func (p *podcastManager) CreateFeed(ctx context.Context, url string) (*model.Podcast, error) {
	feed, err := p.getFeed(ctx, url)
	if err != nil {
		return nil, err
	}

	podcast := model.Podcast{
		Url:         url,
		Title:       feed.Title,
		Description: feed.Description,
		ImageUrl:    feed.Image.URL,
	}

	err = p.ds.Podcast(ctx).PutInternal(&podcast)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(podcast.AbsolutePath(), os.ModePerm)
	if err != nil {
		return nil, err
	}

	var episodeErrs []error

	for _, episode := range feed.Items {
		err = p.addPodcastEpisode(ctx, podcast.ID, episode)
		if err != nil {
			episodeErrs = append(episodeErrs, err)
			continue
		}
	}

	if len(episodeErrs) > 0 {
		allErrors := errors.Join(episodeErrs...)
		podcast.Error = allErrors.Error()
		podcast.State = consts.PodcastStatusError
		return &podcast, allErrors
	}

	return &podcast, nil
}

func (p *podcastManager) DeletePodcast(ctx context.Context, id string) error {
	select {
	case p.del <- deleteReq{id: id, isPodcast: true}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *podcastManager) DeletePodcastEpisode(ctx context.Context, id string) error {
	select {
	case p.del <- deleteReq{id: id, isPodcast: false}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *podcastManager) Download(ctx context.Context, id string) error {
	select {
	case p.ch <- id:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *podcastManager) Refresh(ctx context.Context) error {
	select {
	case p.ch <- "":
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *podcastManager) getFeed(ctx context.Context, url string) (*gofeed.Feed, error) {
	return p.parser.ParseURLWithContext(url, ctx)
}

func (p *podcastManager) addPodcastEpisode(ctx context.Context, channelId string, item *gofeed.Item) error {
	var estimatedSize int64
	var url, suffix string

	for _, enclosure := range item.Enclosures {
		if strings.HasPrefix(enclosure.Type, "audio/") {
			extensions, err := mime.ExtensionsByType(enclosure.Type)

			if len(extensions) == 0 || err != nil {
				continue
			}

			suffix = extensions[0][1:]
			url = enclosure.URL
			size, err := strconv.ParseInt(enclosure.Length, 10, 64)
			if err != nil {
				estimatedSize = 0
			} else {
				estimatedSize = size
			}
		}
	}

	if url == "" {
		return model.ErrNotFound
	}

	var duration float32

	if item.ITunesExt != nil {
		duration = durationToSeconds(item.ITunesExt.Duration)
	}

	episode := model.PodcastEpisode{
		PodcastId:   channelId,
		Guid:        item.GUID,
		Url:         url,
		Description: item.Description,
		Title:       item.Title,
		State:       consts.PodcastStatusSkipped,
		ImageUrl:    item.Image.URL,
		PublishDate: item.PublishedParsed,
		Duration:    duration,
		Suffix:      suffix,
		Size:        estimatedSize,
	}

	return p.ds.PodcastEpisode(ctx).Put(&episode)
}

func (p *podcastManager) processDeletes() {
	ctx := context.Background()

	for data := range p.del {
		var err error
		if data.isPodcast {
			err = p.deletePodcast(ctx, data.id)
		} else {
			err = p.deleteEpisode(ctx, data.id)
		}
		if err != nil {
			log.Error(ctx, "failed to process delete", "podcast", data.isPodcast, "id", data.id, "error", err)
		}
	}
}

func (p *podcastManager) deletePodcast(ctx context.Context, id string) error {
	podcast, err := p.ds.Podcast(ctx).Get(id, false)
	if err != nil {
		return err
	}

	var errs []error

	unlinkErr := os.RemoveAll(podcast.AbsolutePath())
	if unlinkErr != nil {
		errs = append(errs, unlinkErr)
	}

	delErr := p.ds.Podcast(ctx).DeleteInternal(id)
	if delErr != nil {
		errs = append(errs, delErr)
	}

	podErr := p.ds.Podcast(ctx).Cleanup()
	if podErr != nil {
		errs = append(errs, podErr)
	}

	epErr := p.ds.PodcastEpisode(ctx).Cleanup()
	if epErr != nil {
		errs = append(errs, epErr)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (p *podcastManager) deleteEpisode(ctx context.Context, id string) error {
	episode, err := p.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}

	if episode.State != consts.PodcastStatusCompleted {
		return nil
	}

	unlinkErr := os.RemoveAll(episode.AbsolutePath())

	episode.State = consts.PodcastStatusDeleted
	delErr := p.ds.PodcastEpisode(ctx).Put(episode)

	if unlinkErr != nil || delErr != nil {
		return errors.Join(unlinkErr, delErr)
	}

	return nil
}

func (p *podcastManager) processFetch() {
	ctx := context.Background()

	for episodeId := range p.ch {
		if episodeId == "" {
			p.refreshPodcasts(ctx)
		} else {
			err := p.downloadPodcast(ctx, episodeId)
			if err != nil {
				log.Error(ctx, "Failed to download podcast episode", "id", episodeId, "error", err)
			}
		}
	}
}

func (p *podcastManager) refreshPodcasts(ctx context.Context) {
	podcasts, err := p.ds.Podcast(ctx).GetAll(false)
	if err != nil {
		log.Error(ctx, "failed to fetch all podcasts", "error", err)
		return
	}

	for i := range podcasts {
		podcast := &podcasts[i]
		err := p.refreshPodcast(ctx, podcast)

		if err != nil {
			podcast.Error = err.Error()
			podcast.State = consts.PodcastStatusError
			updErr := p.ds.Podcast(ctx).PutInternal(podcast)
			log.Error(ctx, "Failed to refresh podcast", "id", podcast.ID, "error", err)

			if updErr != nil {
				log.Error(ctx, "Failed to update podcast error", "id", podcast.ID, "error", updErr)
			}
		}
	}
}

func (p *podcastManager) refreshPodcast(ctx context.Context, podcast *model.Podcast) error {
	feed, err := p.getFeed(ctx, podcast.Url)
	if err != nil {
		return err
	}

	existing, err := p.ds.PodcastEpisode(ctx).GetEpisodeGuids(podcast.ID)
	if err != nil {
		return err
	}

	var episodeErrs []error

	for _, episode := range feed.Items {
		if _, ok := existing[episode.GUID]; ok {
			continue
		}

		err = p.addPodcastEpisode(ctx, podcast.ID, episode)
		if err != nil {
			episodeErrs = append(episodeErrs, err)
			continue
		}
	}

	if len(episodeErrs) > 0 {
		return errors.Join(episodeErrs...)
	}

	return nil
}

func (p *podcastManager) downloadPodcast(ctx context.Context, id string) error {
	episode, err := p.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}

	episode.State = consts.PodcastStatusDownloading
	updErr := p.ds.PodcastEpisode(ctx).Put(episode)
	if updErr != nil {
		return updErr
	}

	defer func() {
		if err != nil {
			episode.Error = err.Error()
			episode.State = consts.PodcastStatusError
			err := p.ds.PodcastEpisode(ctx).Put(episode)
			if err != nil {
				log.Error(ctx, "failed to update to error status", "episode", episode.ID, "error", err)
			}
		}
	}()

	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, episode.Url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", userAgent)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	filepath := episode.AbsolutePath()
	var file *os.File
	file, err = os.Create(filepath)
	if err != nil {
		return err
	}

	defer file.Close()
	if _, err = io.Copy(file, resp.Body); err != nil {
		return err
	}

	var tags map[string]metadata.Tags
	tags, err = metadata.Extract(filepath)
	if err != nil {
		return err
	}

	metadata := tags[filepath]

	episode.Size = metadata.Size()
	episode.Duration = metadata.Duration()
	episode.BitRate = metadata.BitRate()
	episode.Suffix = metadata.Suffix()
	episode.State = consts.PodcastStatusCompleted

	err = p.ds.PodcastEpisode(ctx).Put(episode)
	return err
}

func durationToSeconds(timeString string) float32 {
	durationFloat, err := strconv.ParseFloat(timeString, 64)
	if err == nil {
		return float32(durationFloat)
	}

	splitTime := strings.Split(timeString, ":")
	splitLen := len(splitTime)
	if splitLen < 2 || splitLen > 3 {
		return 0
	}

	duration := float32(0)
	for _, component := range splitTime {
		val, err := strconv.Atoi(component)
		if err != nil {
			return 0
		}

		duration += (60 * duration) + float32(val)
	}

	return duration
}
