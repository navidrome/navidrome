package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaAnnotationController", func() {
	var router *Router
	var ds model.DataStore
	var playTracker *fakePlayTracker
	var eventBroker *fakeEventBroker
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{}
		playTracker = &fakePlayTracker{}
		eventBroker = &fakeEventBroker{}
		router = New(ds, nil, nil, nil, nil, nil, nil, eventBroker, nil, playTracker, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	Describe("Scrobble", func() {
		It("submit all scrobbles with only the id", func() {
			// Back-date the baseline so the assertion still passes on platforms
			// with millisecond clock resolution (e.g. Windows).
			submissionTime := time.Now().Add(-time.Second)
			r := newGetRequest("id=12", "id=34")

			_, err := router.Scrobble(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.Submissions).To(HaveLen(2))
			Expect(playTracker.Submissions[0].Timestamp).To(BeTemporally(">", submissionTime))
			Expect(playTracker.Submissions[0].TrackID).To(Equal("12"))
			Expect(playTracker.Submissions[1].Timestamp).To(BeTemporally(">", submissionTime))
			Expect(playTracker.Submissions[1].TrackID).To(Equal("34"))
		})

		It("submit all scrobbles with respective times", func() {
			time1 := time.Now().Add(-20 * time.Minute)
			t1 := time1.UnixMilli()
			time2 := time.Now().Add(-10 * time.Minute)
			t2 := time2.UnixMilli()
			r := newGetRequest("id=12", "id=34", fmt.Sprintf("time=%d", t1), fmt.Sprintf("time=%d", t2))

			_, err := router.Scrobble(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.Submissions).To(HaveLen(2))
			Expect(playTracker.Submissions[0].Timestamp).To(BeTemporally("~", time1))
			Expect(playTracker.Submissions[0].TrackID).To(Equal("12"))
			Expect(playTracker.Submissions[1].Timestamp).To(BeTemporally("~", time2))
			Expect(playTracker.Submissions[1].TrackID).To(Equal("34"))
		})

		It("checks if number of ids match number of times", func() {
			r := newGetRequest("id=12", "id=34", "time=1111")

			_, err := router.Scrobble(r)

			Expect(err).To(HaveOccurred())
			Expect(playTracker.Submissions).To(BeEmpty())
		})

		Context("with a mix of music and podcast episode ids", func() {
			var notifier *fakePodcastNotifier

			BeforeEach(func() {
				notifier = &fakePodcastNotifier{}
				router.podcastNotifier = notifier
				_ = ds.PodcastChannel(ctx).Put(&model.PodcastChannel{ID: "ch1", Title: "My Channel"})
				_ = ds.PodcastEpisode(ctx).Put(&model.PodcastEpisode{ID: "ep1", ChannelID: "ch1", Title: "Episode One"})
			})

			It("routes the podcast id to the podcast notifier instead of the music scrobbler", func() {
				r := newGetRequest("id=12", "id=ep1")

				_, err := router.Scrobble(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Submissions).To(HaveLen(1))
				Expect(playTracker.Submissions[0].TrackID).To(Equal("12"))
				Expect(notifier.Played).To(HaveLen(1))
				Expect(notifier.Played[0].Episode.ID).To(Equal("ep1"))
				Expect(notifier.Played[0].ChannelTitle).To(Equal("My Channel"))
			})

			It("submits nothing to the music scrobbler when every id is a podcast episode", func() {
				r := newGetRequest("id=ep1")

				_, err := router.Scrobble(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Submissions).To(BeEmpty())
				Expect(notifier.Played).To(HaveLen(1))
			})
		})

		Context("submission=false", func() {
			var req *http.Request
			BeforeEach(func() {
				_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "12"})
				ctx = request.WithPlayer(ctx, model.Player{ID: "player-1"})
				req = newGetRequest("id=12", "submission=false")
				req = req.WithContext(ctx)
			})

			It("does not scrobble", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Submissions).To(BeEmpty())
			})

			It("registers a NowPlaying via ReportPlayback", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.ReportedPlayback).To(HaveLen(1))
				Expect(playTracker.ReportedPlayback[0].MediaId).To(Equal("12"))
				Expect(playTracker.ReportedPlayback[0].State).To(Equal(scrobbler.StatePlaying))
				Expect(playTracker.ReportedPlayback[0].ClientId).To(Equal("player-1"))
			})
		})
	})

	Describe("UserTags (AI) / MyTags (human) source isolation", func() {
		var tagRepo *fakeMediaFileTagRepo

		BeforeEach(func() {
			tagRepo = &fakeMediaFileTagRepo{}
			ds.(*tests.MockDataStore).MockedMediaFileTag = tagRepo

			_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "ai-song"})
			_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "user-song"})

			// Same tag name ("rock") applied by both sources, on different songs -
			// exercises that neither family leaks into the other's results.
			_ = tagRepo.TagSong("ai-song", "rock", model.MediaFileTagSourceAI)
			_ = tagRepo.TagSong("user-song", "rock", model.MediaFileTagSourceUser)
			_ = tagRepo.TagSong("user-song", "workout", model.MediaFileTagSourceUser)
		})

		It("GetAllUserTags only returns AI-sourced tag names", func() {
			r := newGetRequest()
			resp, err := router.GetAllUserTags(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.UserTags.Tag).To(ConsistOf("rock"))
		})

		It("GetAllMyTags only returns user-sourced tag names", func() {
			r := newGetRequest()
			resp, err := router.GetAllMyTags(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.UserTags).To(BeNil())
			Expect(resp.MyTags.Tag).To(ConsistOf("rock", "workout"))
		})

		It("GetSongsByUserTag for a name used in both sources only returns the AI-tagged song", func() {
			r := newGetRequest("tag=rock")
			resp, err := router.GetSongsByUserTag(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SongsByUserTag.Songs).To(HaveLen(1))
			Expect(resp.SongsByUserTag.Songs[0].Id).To(Equal("ai-song"))
		})

		It("GetSongsByMyTag for a name used in both sources only returns the user-tagged song", func() {
			r := newGetRequest("tag=rock")
			resp, err := router.GetSongsByMyTag(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SongsByMyTag).ToNot(BeNil())
			Expect(resp.SongsByMyTag.Songs).To(HaveLen(1))
			Expect(resp.SongsByMyTag.Songs[0].Id).To(Equal("user-song"))
		})

		It("GetAllMyTags returns a valid empty response, not an error, when there are no user tags", func() {
			tagRepo.tags = nil
			r := newGetRequest()
			resp, err := router.GetAllMyTags(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.MyTags).ToNot(BeNil())
			Expect(resp.MyTags.Tag).To(BeEmpty())
		})

		It("GetSongsByMyTag returns a valid empty response, not an error, for an unknown tag", func() {
			r := newGetRequest("tag=does-not-exist")
			resp, err := router.GetSongsByMyTag(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SongsByMyTag).ToNot(BeNil())
			Expect(resp.SongsByMyTag.Songs).To(BeEmpty())
		})

		It("GetSongsByMyTag requires the tag parameter", func() {
			r := newGetRequest()
			_, err := router.GetSongsByMyTag(r)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ReportPlayback", func() {
		It("returns error when mediaId is missing", func() {
			r := newGetRequest("mediaType=song", "positionMs=0", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when mediaType is missing", func() {
			r := newGetRequest("mediaId=123", "positionMs=0", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when positionMs is missing", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when state is missing", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid state value", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=invalid")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for negative positionMs", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=-1", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for NaN playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=NaN")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for Inf playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=Inf")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for negative playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=-1.0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for zero playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("accepts mediaType=podcast without error", func() {
			r := newGetRequest("mediaId=123", "mediaType=podcast", "positionMs=0", "state=playing")
			ctx := request.WithPlayer(r.Context(), model.Player{ID: "p1"})
			r = r.WithContext(ctx)
			resp, err := router.ReportPlayback(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("defaults playbackRate to 1.0 and ignoreScrobble to false", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=5000", "state=playing")
			ctx := request.WithPlayer(r.Context(), model.Player{ID: "p1"})
			r = r.WithContext(ctx)
			_, err := router.ReportPlayback(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.ReportedPlayback).To(HaveLen(1))
			Expect(playTracker.ReportedPlayback[0].PlaybackRate).To(Equal(1.0))
			Expect(playTracker.ReportedPlayback[0].IgnoreScrobble).To(BeFalse())
			Expect(playTracker.ReportedPlayback[0].ClientId).To(Equal("p1"))
			Expect(playTracker.ReportedPlayback[0].ClientName).To(BeEmpty())
		})
	})

	Describe("Star/Unstar playlists", func() {
		var plRepo *tests.MockPlaylistRepo

		BeforeEach(func() {
			plRepo = tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "pl-1", Name: "My Playlist", OwnerID: "u1"}})
			ds.(*tests.MockDataStore).MockedPlaylist = plRepo
		})

		It("stars a playlist by dispatching to the Playlist repo", func() {
			r := newGetRequest("id=pl-1")

			_, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(plRepo.Starred).To(HaveKeyWithValue("pl-1", true))
		})

		It("unstars a playlist by dispatching to the Playlist repo", func() {
			r := newGetRequest("id=pl-1")

			_, err := router.Unstar(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(plRepo.Starred).To(HaveKeyWithValue("pl-1", false))
		})
	})

	Describe("SetRating playlists", func() {
		var plRepo *tests.MockPlaylistRepo

		BeforeEach(func() {
			plRepo = tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "pl-1", Name: "My Playlist", OwnerID: "u1"}})
			ds.(*tests.MockDataStore).MockedPlaylist = plRepo
		})

		It("rates a playlist by dispatching to the Playlist repo", func() {
			r := newGetRequest("id=pl-1", "rating=4")

			_, err := router.SetRating(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(plRepo.Ratings).To(HaveKeyWithValue("pl-1", 4))
		})
	})

	Describe("Star with an unresolvable id", func() {
		It("skips the id without broadcasting an empty (wildcard) refresh", func() {
			r := newGetRequest("id=does-not-exist")

			_, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(eventBroker.Events).To(BeEmpty())
		})
	})
})

type fakePlayTracker struct {
	Submissions      []scrobbler.Submission
	ReportedPlayback []scrobbler.ReportPlaybackParams
	Error            error
}

func (f *fakePlayTracker) GetNowPlaying(_ context.Context) ([]scrobbler.PlaybackSession, error) {
	return nil, f.Error
}

func (f *fakePlayTracker) Submit(_ context.Context, submissions []scrobbler.Submission) error {
	if f.Error != nil {
		return f.Error
	}
	f.Submissions = append(f.Submissions, submissions...)
	return nil
}

func (f *fakePlayTracker) ReportPlayback(_ context.Context, params scrobbler.ReportPlaybackParams) error {
	if f.Error != nil {
		return f.Error
	}
	f.ReportedPlayback = append(f.ReportedPlayback, params)
	return nil
}

var _ scrobbler.PlayTracker = (*fakePlayTracker)(nil)

type podcastPlayedCall struct {
	Username     string
	PlayerName   string
	Source       string
	Episode      *model.PodcastEpisode
	ChannelTitle string
}

type fakePodcastNotifier struct {
	Played []podcastPlayedCall
}

func (f *fakePodcastNotifier) DispatchPodcastPlayed(_ context.Context, username, playerName, source string, episode *model.PodcastEpisode, channelTitle string) {
	f.Played = append(f.Played, podcastPlayedCall{
		Username:     username,
		PlayerName:   playerName,
		Source:       source,
		Episode:      episode,
		ChannelTitle: channelTitle,
	})
}

var _ PodcastPlayNotifier = (*fakePodcastNotifier)(nil)

type fakeEventBroker struct {
	http.Handler
	Events []events.Event
}

func (f *fakeEventBroker) SendMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

func (f *fakeEventBroker) SendBroadcastMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

var _ events.Broker = (*fakeEventBroker)(nil)

// fakeMediaFileTagRepo is an in-memory model.MediaFileTagRepository double,
// used to verify the UserTag (AI) / MyTag (human) Subsonic endpoint families
// never leak into each other's results.
type fakeMediaFileTagRepo struct {
	tags []model.MediaFileTag
}

func (f *fakeMediaFileTagRepo) TagSong(mediaFileID, tagName, source string) error {
	f.tags = append(f.tags, model.MediaFileTag{MediaFileID: mediaFileID, TagName: tagName, Source: source})
	return nil
}

func (f *fakeMediaFileTagRepo) UntagSong(mediaFileID, tagName string) error {
	filtered := f.tags[:0]
	for _, t := range f.tags {
		if t.MediaFileID == mediaFileID && t.TagName == tagName {
			continue
		}
		filtered = append(filtered, t)
	}
	f.tags = filtered
	return nil
}

func (f *fakeMediaFileTagRepo) TagsForSong(mediaFileID, source string) ([]string, error) {
	var names []string
	for _, t := range f.tags {
		if t.MediaFileID == mediaFileID && (source == "" || t.Source == source) {
			names = append(names, t.TagName)
		}
	}
	return names, nil
}

func (f *fakeMediaFileTagRepo) AllTagNames(source string) ([]string, error) {
	seen := map[string]bool{}
	var names []string
	for _, t := range f.tags {
		if source != "" && t.Source != source {
			continue
		}
		if !seen[t.TagName] {
			seen[t.TagName] = true
			names = append(names, t.TagName)
		}
	}
	return names, nil
}

func (f *fakeMediaFileTagRepo) SongIDsForTag(tagName, source string) ([]string, error) {
	var ids []string
	for _, t := range f.tags {
		if t.TagName == tagName && (source == "" || t.Source == source) {
			ids = append(ids, t.MediaFileID)
		}
	}
	return ids, nil
}

func (f *fakeMediaFileTagRepo) TagCounts(_ string) ([]model.TagCount, error) {
	return nil, nil
}

var _ model.MediaFileTagRepository = (*fakeMediaFileTagRepo)(nil)
