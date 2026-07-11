package subsonic

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetPodcasts(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	includeEpisodes := p.BoolOr("includeEpisodes", true)
	id, _ := p.String("id")

	var channels model.PodcastChannels
	if id != "" {
		ch, err := api.ds.PodcastChannel(ctx).GetWithEpisodes(id)
		if err != nil {
			return nil, err
		}
		channels = model.PodcastChannels{*ch}
	} else {
		all, err := api.ds.PodcastChannel(ctx).GetAll(model.QueryOptions{Sort: "title"})
		if err != nil {
			return nil, err
		}
		if includeEpisodes {
			for i := range all {
				full, err := api.ds.PodcastChannel(ctx).GetWithEpisodes(all[i].ID)
				if err != nil {
					log.Warn(ctx, "Error loading episodes for podcast channel", "id", all[i].ID, err)
					continue
				}
				all[i] = *full
			}
		}
		channels = all
	}

	res := make([]responses.PodcastChannel, len(channels))
	for i, ch := range channels {
		res[i] = toPodcastChannel(ch, includeEpisodes)
	}

	response := newResponse()
	response.Podcasts = &responses.Podcasts{Channel: res}
	return response, nil
}

func (api *Router) GetNewestPodcasts(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	count := p.IntOr("count", 20)

	episodes, err := api.ds.PodcastEpisode(ctx).GetNewest(count)
	if err != nil {
		return nil, err
	}

	channelCache := map[string]model.PodcastChannel{}
	res := make([]responses.PodcastEpisode, 0, len(episodes))
	for _, ep := range episodes {
		channel, ok := channelCache[ep.ChannelID]
		if !ok {
			ch, err := api.ds.PodcastChannel(ctx).Get(ep.ChannelID)
			if err != nil {
				log.Warn(ctx, "Error loading channel for newest podcast episode", "channel", ep.ChannelID, err)
				continue
			}
			channel = *ch
			channelCache[ep.ChannelID] = channel
		}
		res = append(res, toPodcastEpisode(ep, channel))
	}

	response := newResponse()
	response.NewestPodcasts = &responses.NewestPodcasts{Episode: res}
	return response, nil
}

// RefreshPodcasts is fire-and-forget per the Subsonic spec: it kicks off a
// refresh of all subscriptions and returns immediately, detached from the
// request context so it survives the response being sent.
func (api *Router) RefreshPodcasts(r *http.Request) (*responses.Subsonic, error) {
	bgCtx := context.WithoutCancel(r.Context())
	go func() {
		if err := api.podcasts.RefreshAll(bgCtx); err != nil {
			log.Error(bgCtx, "Error refreshing podcasts", err)
		}
	}()
	return newResponse(), nil
}

func (api *Router) CreatePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	url, err := p.String("url")
	if err != nil {
		return nil, err
	}
	if _, err := api.podcasts.CreateChannel(r.Context(), url); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeletePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if err := api.podcasts.DeleteChannel(r.Context(), id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeletePodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if err := api.podcasts.DeleteEpisode(r.Context(), id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// DownloadPodcastEpisode is fire-and-forget per the Subsonic spec: it
// enqueues the download and returns immediately without waiting for it to
// complete, detached from the request context.
func (api *Router) DownloadPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	bgCtx := context.WithoutCancel(r.Context())
	go func() {
		if err := api.podcasts.DownloadEpisode(bgCtx, id); err != nil {
			log.Error(bgCtx, "Error downloading podcast episode", "id", id, err)
		}
	}()
	return newResponse(), nil
}

// podcastEpisodeStatus maps our internal download-status vocabulary to the
// Subsonic spec's episode status attribute.
func podcastEpisodeStatus(status model.PodcastEpisodeDownloadStatus, policy model.PodcastDownloadPolicy) string {
	switch status {
	case model.PodcastEpisodeQueued, model.PodcastEpisodeDownloading:
		return "downloading"
	case model.PodcastEpisodeDownloaded:
		return "completed"
	case model.PodcastEpisodeDownloadError:
		return "error"
	case model.PodcastEpisodeDeleted:
		return "deleted"
	case model.PodcastEpisodeNotDownloaded:
		if policy == model.PodcastDownloadPolicyNone {
			return "skipped"
		}
		return "new"
	default:
		return "new"
	}
}

func toPodcastChannel(ch model.PodcastChannel, includeEpisodes bool) responses.PodcastChannel {
	var coverArt string
	if ch.UploadedImage != "" {
		coverArt = ch.CoverArtID().String()
	}
	res := responses.PodcastChannel{
		Id:               ch.ID,
		Url:              ch.Url,
		Title:            ch.Title,
		Description:      ch.Description,
		CoverArt:         coverArt,
		OriginalImageUrl: ch.CoverArtUrl,
		Status:           string(ch.Status),
		ErrorMessage:     ch.ErrorMessage,
	}
	if includeEpisodes {
		res.Episode = make([]responses.PodcastEpisode, len(ch.Episodes))
		for i, ep := range ch.Episodes {
			res.Episode[i] = toPodcastEpisode(ep, ch)
		}
	}
	return res
}

// toPodcastEpisode always populates streamId (regardless of download
// status), since stream.go's proxy fallback makes every episode streamable
// - this is what makes stream-only channels usable by Subsonic clients.
func toPodcastEpisode(ep model.PodcastEpisode, channel model.PodcastChannel) responses.PodcastEpisode {
	var coverArt string
	if channel.UploadedImage != "" {
		coverArt = channel.CoverArtID().String()
	}
	return responses.PodcastEpisode{
		Child: responses.Child{
			Id:          ep.ID,
			Parent:      channel.ID,
			IsDir:       false,
			Title:       ep.Title,
			Album:       channel.Title,
			Artist:      channel.Title,
			CoverArt:    coverArt,
			Size:        ep.Size,
			ContentType: ep.ContentType,
			Suffix:      ep.Suffix,
			Duration:    int32(ep.Duration),
			BitRate:     int32(ep.BitRate),
		},
		StreamId:    ep.ID,
		ChannelId:   ep.ChannelID,
		Description: ep.Description,
		Status:      podcastEpisodeStatus(ep.DownloadStatus, channel.DownloadPolicy),
		PublishDate: ep.PublishDate,
	}
}
