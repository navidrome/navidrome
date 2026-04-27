package subsonic

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetPodcasts(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id := p.StringOr("id", "")
	includeEpisodes := p.BoolOr("includeEpisodes", true)

	ctx := r.Context()
	chRepo := api.ds.PodcastChannel(ctx)

	var channels model.PodcastChannels
	var err error

	if id != "" {
		ch, e := chRepo.Get(id)
		if e != nil {
			return nil, e
		}
		channels = model.PodcastChannels{*ch}
	} else {
		channels, err = chRepo.GetAll(includeEpisodes)
		if err != nil {
			return nil, err
		}
	}

	if includeEpisodes && id != "" {
		eps, e := api.ds.PodcastEpisode(ctx).GetByChannel(id)
		if e != nil {
			return nil, e
		}
		channels[0].Episodes = eps
	}

	resp := newResponse()
	resp.Podcasts = &responses.Podcasts{}
	for _, ch := range channels {
		rch := responses.PodcastChannel{
			ID:               ch.ID,
			URL:              ch.URL,
			Title:            ch.Title,
			Description:      ch.Description,
			OriginalImageUrl: ch.ImageURL,
			Status:           string(ch.Status),
			ErrorMessage:     ch.ErrorMessage,
		}
		if includeEpisodes {
			for _, ep := range ch.Episodes {
				rch.Episode = append(rch.Episode, buildPodcastEpisode(ep))
			}
		}
		resp.Podcasts.Channel = append(resp.Podcasts.Channel, rch)
	}
	return resp, nil
}

func (api *Router) GetNewestPodcasts(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	count := p.IntOr("count", 20)

	eps, err := api.ds.PodcastEpisode(r.Context()).GetNewest(count)
	if err != nil {
		return nil, err
	}

	resp := newResponse()
	resp.NewestPodcasts = &responses.NewestPodcasts{}
	for _, ep := range eps {
		child := responses.Child{
			Id:          ep.ID,
			Title:       ep.Title,
			IsDir:       false,
			Parent:      ep.ChannelID,
			Duration:    int32(ep.Duration),
			Size:        ep.Size,
			BitRate:     int32(ep.BitRate),
			Suffix:      ep.Suffix,
			ContentType: ep.ContentType,
			Type:        "podcast",
			ChannelId:   ep.ChannelID,
			Description: ep.Description,
			Status:      string(ep.Status),
		}
		if !ep.PublishDate.IsZero() {
			child.PublishDate = ep.PublishDate.UTC().Format(time.RFC3339)
		}
		resp.NewestPodcasts.Episode = append(resp.NewestPodcasts.Episode, child)
	}
	return resp, nil
}

func (api *Router) CreatePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	if err := requireAdmin(r); err != nil {
		return nil, err
	}
	p := req.Params(r)
	feedURL, err := p.String("url")
	if err != nil {
		return nil, err
	}
	if err := api.podcasts.AddChannel(r.Context(), feedURL); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) RefreshPodcasts(r *http.Request) (*responses.Subsonic, error) {
	if err := requireAdmin(r); err != nil {
		return nil, err
	}
	if err := api.podcasts.RefreshChannels(r.Context()); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeletePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	if err := requireAdmin(r); err != nil {
		return nil, err
	}
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
	if err := requireAdmin(r); err != nil {
		return nil, err
	}
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

func (api *Router) DownloadPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	if err := requireAdmin(r); err != nil {
		return nil, err
	}
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if err := api.podcasts.DownloadEpisode(r.Context(), id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	ep, err := api.ds.PodcastEpisode(r.Context()).Get(id)
	if err != nil {
		return nil, err
	}
	resp := newResponse()
	re := buildPodcastEpisode(*ep)
	resp.PodcastEpisode = &re
	return resp, nil
}

func buildPodcastEpisode(ep model.PodcastEpisode) responses.PodcastEpisode {
	re := responses.PodcastEpisode{
		ID:              ep.ID,
		StreamId:        ep.StreamID,
		ChannelId:       ep.ChannelID,
		Title:           ep.Title,
		Description:     ep.Description,
		Status:          string(ep.Status),
		ErrorMessage:    ep.ErrorMessage,
		Duration:        ep.Duration,
		Size:            ep.Size,
		Suffix:          ep.Suffix,
		ContentType:     ep.ContentType,
		BitRate:         ep.BitRate,
		DownloadedBytes: ep.DownloadedBytes,
	}
	if !ep.PublishDate.IsZero() {
		re.PublishDate = ep.PublishDate.UTC().Format(time.RFC3339)
	}
	return re
}

func requireAdmin(r *http.Request) error {
	user, ok := request.UserFrom(r.Context())
	if !ok || !user.IsAdmin {
		return newError(responses.ErrorAuthorizationFail)
	}
	return nil
}
