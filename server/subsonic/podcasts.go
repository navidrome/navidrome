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

	// Collect channel IDs for bulk queries.
	channelIDs := make([]string, len(channels))
	for i, ch := range channels {
		channelIDs[i] = ch.ID
	}

	// Load channel persons
	personRepo := api.ds.PodcastPerson(ctx)
	channelPersons := make(map[string]model.PodcastPersons)
	for _, ch := range channels {
		persons, err := personRepo.GetByChannel(ch.ID)
		if err == nil {
			channelPersons[ch.ID] = persons
		}
	}

	// Bulk-load podcast:podroll items
	podrollRepo := api.ds.PodcastPodroll(ctx)
	allPodrolls, _ := podrollRepo.GetByChannels(channelIDs)
	podrollMap := make(map[string]model.PodcastPodrollItems)
	for _, pr := range allPodrolls {
		podrollMap[pr.ChannelID] = append(podrollMap[pr.ChannelID], pr)
	}

	// Load podcast:liveItem per channel
	liveItemRepo := api.ds.PodcastLiveItem(ctx)
	liveItemMap := make(map[string]*model.PodcastLiveItem)
	for _, chID := range channelIDs {
		if li, err := liveItemRepo.GetByChannel(chID); err == nil {
			liveItemMap[chID] = li
		}
	}

	// Bulk-load episode transcripts and persons when including episodes
	var epTranscripts map[string]model.PodcastTranscripts
	var epPersons map[string]model.PodcastPersons
	if includeEpisodes {
		var epIDs []string
		for _, ch := range channels {
			for _, ep := range ch.Episodes {
				epIDs = append(epIDs, ep.ID)
			}
		}
		if len(epIDs) > 0 {
			transcriptRepo := api.ds.PodcastTranscript(ctx)
			allTranscripts, err := transcriptRepo.GetByEpisodes(epIDs)
			if err == nil {
				epTranscripts = make(map[string]model.PodcastTranscripts)
				for _, t := range allTranscripts {
					epTranscripts[t.EpisodeID] = append(epTranscripts[t.EpisodeID], t)
				}
			}
			allPersons, err := personRepo.GetByEpisodes(epIDs)
			if err == nil {
				epPersons = make(map[string]model.PodcastPersons)
				for _, p := range allPersons {
					epPersons[p.EpisodeID] = append(epPersons[p.EpisodeID], p)
				}
			}
		}
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
			// Podcasting 2.0 Tier 1 & 2
			PodcastGuid:     ch.PodcastGUID,
			Locked:          ch.Locked,
			Medium:          ch.Medium,
			FundingUrl:      ch.FundingURL,
			FundingText:     ch.FundingText,
			UpdateFrequency: ch.UpdateFrequency,
			Complete:        ch.Complete,
			// Podcasting 2.0 Tier 3
			UsesPodping: ch.UsesPodping,
		}
		for _, p := range channelPersons[ch.ID] {
			rch.Person = append(rch.Person, responses.PodcastPersonResp{
				Name:  p.Name,
				Role:  p.Role,
				Group: p.Group,
				Img:   p.Img,
				Href:  p.Href,
			})
		}
		for _, pr := range podrollMap[ch.ID] {
			rch.Podroll = append(rch.Podroll, responses.PodcastPodrollResp{
				FeedGUID: pr.FeedGUID,
				FeedURL:  pr.FeedURL,
				Title:    pr.Title,
			})
		}
		if li := liveItemMap[ch.ID]; li != nil {
			liveResp := &responses.PodcastLiveItemResp{
				Status:          li.Status,
				Title:           li.Title,
				GUID:            li.GUID,
				EnclosureURL:    li.EnclosureURL,
				EnclosureType:   li.EnclosureType,
				ContentLinkURL:  li.ContentLinkURL,
				ContentLinkText: li.ContentLinkText,
			}
			if !li.StartTime.IsZero() {
				liveResp.StartTime = li.StartTime.UTC().Format(time.RFC3339)
			}
			if !li.EndTime.IsZero() {
				liveResp.EndTime = li.EndTime.UTC().Format(time.RFC3339)
			}
			rch.LiveItem = liveResp
		}
		if includeEpisodes {
			for _, ep := range ch.Episodes {
				ep.Transcripts = epTranscripts[ep.ID]
				ep.Persons = epPersons[ep.ID]
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
	ctx := r.Context()
	ep, err := api.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	ep.Transcripts, _ = api.ds.PodcastTranscript(ctx).GetByEpisode(ep.ID)
	ep.Persons, _ = api.ds.PodcastPerson(ctx).GetByEpisode(ep.ID)

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
		// Podcasting 2.0
		Season:         ep.Season,
		SeasonName:     ep.SeasonName,
		EpisodeNumber:  ep.EpisodeNumber,
		EpisodeDisplay: ep.EpisodeDisplay,
		ChaptersUrl:    ep.ChaptersURL,
		SoundbiteStart: ep.SoundbiteStart,
		SoundbiteDur:   ep.SoundbiteDur,
	}
	if !ep.PublishDate.IsZero() {
		re.PublishDate = ep.PublishDate.UTC().Format(time.RFC3339)
	}
	for _, t := range ep.Transcripts {
		re.Transcript = append(re.Transcript, responses.PodcastTranscriptResp{
			URL:      t.URL,
			Type:     t.MimeType,
			Language: t.Language,
			Rel:      t.Rel,
		})
	}
	for _, p := range ep.Persons {
		re.Person = append(re.Person, responses.PodcastPersonResp{
			Name:  p.Name,
			Role:  p.Role,
			Group: p.Group,
			Img:   p.Img,
			Href:  p.Href,
		})
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
