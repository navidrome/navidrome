package subsonic

import (
	"net/http"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) CreatePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	if conf.Server.Podcast.AdminOnly {
		user := getUser(ctx)
		if !user.IsAdmin {
			return nil, newError(responses.ErrorAuthorizationFail, "Creating podcasts is admin only")
		}
	}

	p := req.Params(r)
	url, err := p.String("url")
	if err != nil {
		return nil, err
	}

	_, err = api.podcastManager.CreateFeed(ctx, url)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	return response, nil
}

func (api *Router) DeletePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	if conf.Server.Podcast.AdminOnly {
		user := getUser(ctx)
		if !user.IsAdmin {
			return nil, newError(responses.ErrorAuthorizationFail, "Deleting podcasts is admin only")
		}
	}

	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !model.IsPodcastId(id) {
		return nil, rest.ErrNotFound
	}
	id = model.ExtractExternalId(id)

	err = api.podcastManager.DeletePodcast(ctx, id)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	return response, nil
}

func (api *Router) DeletePodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	if conf.Server.Podcast.AdminOnly {
		user := getUser(ctx)
		if !user.IsAdmin {
			return nil, newError(responses.ErrorAuthorizationFail, "Deleting podcast episodes is admin only")
		}
	}

	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !model.IsPodcastEpisodeId(id) {
		return nil, rest.ErrNotFound
	}
	id = model.ExtractExternalId(id)

	err = api.podcastManager.DeletePodcastEpisode(ctx, id)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	return response, nil
}

func (api *Router) DownloadPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	if conf.Server.Podcast.AdminOnly {
		user := getUser(ctx)
		if !user.IsAdmin {
			return nil, newError(responses.ErrorAuthorizationFail, "Downloading podcast episodes is admin only")
		}
	}

	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !model.IsPodcastEpisodeId(id) {
		return nil, rest.ErrNotFound
	}
	id = model.ExtractExternalId(id)

	err = api.podcastManager.Download(ctx, id)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	return response, nil
}

func (api *Router) GetNewestPodcasts(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	count := p.IntOr("count", 20)

	episodes, err := api.ds.PodcastEpisode(ctx).GetNewestEpisodes(count)
	if err != nil {
		return nil, err
	}

	subsonicEpisodes := []responses.PodcastEpisode{}
	for i := range episodes {
		subsonicEp := buildPodcastEpisode(ctx, &episodes[i])
		subsonicEpisodes = append(subsonicEpisodes, subsonicEp)
	}

	response := newResponse()
	response.NewestPodcasts = &responses.NewestPodcasts{
		Episodes: subsonicEpisodes,
	}

	return response, nil
}

func (api *Router) GetPodcasts(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	includeEpisodes := p.BoolOr("includeEpisodes", true)
	id := p.StringOr("id", "")

	options := model.QueryOptions{}
	if id != "" {
		if !model.IsPodcastId(id) {
			return nil, rest.ErrNotFound
		}
		id = model.ExtractExternalId(id)

		options = model.QueryOptions{
			Filters: squirrel.Eq{"id": id},
		}
	}

	podcasts, err := api.ds.Podcast(ctx).GetAll(includeEpisodes, options)
	if err != nil {
		return nil, err
	}

	subsonicPodcasts := []responses.PodcastChannel{}
	for i := range podcasts {
		subsonicPd := buildPodcast(ctx, includeEpisodes, &podcasts[i])
		subsonicPodcasts = append(subsonicPodcasts, subsonicPd)
	}

	response := newResponse()
	response.Podcasts = &responses.Podcasts{
		Podcasts: subsonicPodcasts,
	}

	return response, nil
}

func (api *Router) RefreshPodcasts(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	if conf.Server.Podcast.AdminOnly {
		user := getUser(ctx)
		if !user.IsAdmin {
			return nil, newError(responses.ErrorAuthorizationFail, "Refreshing podcasts is admin only")
		}
	}

	err := api.podcastManager.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	return response, nil
}
