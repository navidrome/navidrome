package podcasts

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
)

func newTestPodcasts() (*podcasts, *tests.MockedPodcastChannelRepo, *tests.MockedPodcastSubscriptionRepo) {
	ds := &tests.MockDataStore{}
	channelRepo := tests.CreateMockedPodcastChannelRepo()
	ds.MockedPodcastChannel = channelRepo
	subRepo := tests.CreateMockedPodcastSubscriptionRepo()
	ds.MockedPodcastSubscription = subRepo
	ds.MockedPodcastEpisode = tests.CreateMockedPodcastEpisodeRepo()
	return New(ds, nil).(*podcasts), channelRepo, subRepo
}

// A non-admin subscriber subscribing to a feed that some other user already subscribes to must
// reuse the existing shared channel row, not create a second one - the whole point of the shared-
// channel model is that a feed is only ever fetched/stored once regardless of subscriber count.
func TestSubscribeReusesExistingSharedChannel(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	channel, err := svc.Subscribe(ctx, existing.Url)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if channel.ID != existing.ID {
		t.Fatalf("expected Subscribe to reuse existing channel %q, got %q", existing.ID, channel.ID)
	}
	if len(channelRepo.Data) != 1 {
		t.Fatalf("expected exactly 1 channel to exist, got %d", len(channelRepo.Data))
	}

	sub, err := subRepo.FindByChannelAndUser(existing.ID, "user-2")
	if err != nil {
		t.Fatalf("expected a subscription to be created for user-2: %v", err)
	}
	if sub.ChannelID != existing.ID {
		t.Fatalf("subscription points at wrong channel: %q", sub.ChannelID)
	}
}

// Subscribing twice for the same user/channel is idempotent - it must not create a second
// subscription row.
func TestSubscribeIsIdempotent(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	if _, err := svc.Subscribe(ctx, existing.Url); err != nil {
		t.Fatalf("first Subscribe returned error: %v", err)
	}
	if _, err := svc.Subscribe(ctx, existing.Url); err != nil {
		t.Fatalf("second Subscribe returned error: %v", err)
	}

	count := 0
	for _, s := range subRepo.Data {
		if s.ChannelID == existing.ID && s.UserID == "user-2" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 subscription for user-2, got %d", count)
	}
}

// Unsubscribing the last remaining subscriber must tear down the now-orphaned shared channel.
func TestUnsubscribeLastSubscriberDeletesChannel(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing
	sub := &model.PodcastSubscription{ID: "sub-1", ChannelID: existing.ID, UserID: "user-2"}
	if err := subRepo.Put(sub); err != nil {
		t.Fatalf("seeding subscription failed: %v", err)
	}

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	if err := svc.Unsubscribe(ctx, existing.ID); err != nil {
		t.Fatalf("Unsubscribe returned error: %v", err)
	}
	if _, ok := channelRepo.Data[existing.ID]; ok {
		t.Fatalf("expected orphaned channel to be deleted")
	}
}
