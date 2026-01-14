package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDiscordPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Discord Plugin Main Suite")
}

var _ = Describe("discordPlugin", func() {
	var plugin discordPlugin

	BeforeEach(func() {
		plugin = discordPlugin{}
		pdk.ResetMock()
		host.CacheMock.ExpectedCalls = nil
		host.CacheMock.Calls = nil
		host.ConfigMock.ExpectedCalls = nil
		host.ConfigMock.Calls = nil
		host.WebSocketMock.ExpectedCalls = nil
		host.WebSocketMock.Calls = nil
		host.SchedulerMock.ExpectedCalls = nil
		host.SchedulerMock.Calls = nil
		host.ArtworkMock.ExpectedCalls = nil
		host.ArtworkMock.Calls = nil
	})

	Describe("getConfig", func() {
		It("returns config values when properly set", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{"user.user1", "user.user2"})
			host.ConfigMock.On("Get", "user.user1").Return("token1", true)
			host.ConfigMock.On("Get", "user.user2").Return("token2", true)
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()

			clientID, users, err := getConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(clientID).To(Equal("test-client-id"))
			Expect(users).To(HaveLen(2))
			Expect(users["user1"]).To(Equal("token1"))
			Expect(users["user2"]).To(Equal("token2"))
		})

		It("returns empty client ID when not set", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("", false)
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()

			clientID, users, err := getConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(clientID).To(BeEmpty())
			Expect(users).To(BeNil())
		})

		It("returns nil users when users not configured", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{})
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()

			clientID, users, err := getConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(clientID).To(Equal("test-client-id"))
			Expect(users).To(BeNil())
		})
	})

	Describe("IsAuthorized", func() {
		BeforeEach(func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
		})

		It("returns true for authorized user", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{"user.testuser"})
			host.ConfigMock.On("Get", "user.testuser").Return("token123", true)

			authorized, err := plugin.IsAuthorized(scrobbler.IsAuthorizedRequest{
				Username: "testuser",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(authorized).To(BeTrue())
		})

		It("returns false for unauthorized user", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{"user.otheruser"})
			host.ConfigMock.On("Get", "user.otheruser").Return("token123", true)

			authorized, err := plugin.IsAuthorized(scrobbler.IsAuthorizedRequest{
				Username: "testuser",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(authorized).To(BeFalse())
		})
	})

	Describe("NowPlaying", func() {
		BeforeEach(func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
		})

		It("returns not authorized error when user not in config", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{"user.otheruser"})
			host.ConfigMock.On("Get", "user.otheruser").Return("token", true)

			err := plugin.NowPlaying(scrobbler.NowPlayingRequest{
				Username: "testuser",
				Track:    scrobbler.TrackInfo{Title: "Test Song"},
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, scrobbler.ScrobblerErrorNotAuthorized)).To(BeTrue())
		})

		It("successfully sends now playing update", func() {
			pdk.PDKMock.On("GetConfig", clientIDKey).Return("test-client-id", true)
			host.ConfigMock.On("Keys", userKeyPrefix).Return([]string{"user.testuser"})
			host.ConfigMock.On("Get", "user.testuser").Return("test-token", true)

			// Connect mocks (isConnected check via heartbeat)
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(0), false, errors.New("not found"))

			// Mock HTTP GET request for gateway discovery
			gatewayResp := []byte(`{"url":"wss://gateway.discord.gg"}`)
			gatewayReq := &pdk.HTTPRequest{}
			pdk.PDKMock.On("NewHTTPRequest", pdk.MethodGet, "https://discord.com/api/gateway").Return(gatewayReq).Once()
			pdk.PDKMock.On("Send", gatewayReq).Return(pdk.NewStubHTTPResponse(200, nil, gatewayResp)).Once()

			// Mock WebSocket connection
			host.WebSocketMock.On("Connect", mock.MatchedBy(func(url string) bool {
				return strings.Contains(url, "gateway.discord.gg")
			}), mock.Anything, "testuser").Return("testuser", nil)
			host.WebSocketMock.On("SendText", "testuser", mock.Anything).Return(nil)
			host.SchedulerMock.On("ScheduleRecurring", mock.Anything, payloadHeartbeat, "testuser").Return("testuser", nil)

			// Cancel existing clear schedule (may or may not exist)
			host.SchedulerMock.On("CancelSchedule", "testuser-clear").Return(nil)

			// Image mocks - cache miss, will make HTTP request to Discord
			host.CacheMock.On("GetString", mock.MatchedBy(func(key string) bool {
				return strings.HasPrefix(key, "discord.image.")
			})).Return("", false, nil)
			host.CacheMock.On("SetString", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			host.ArtworkMock.On("GetTrackUrl", "track1", int32(300)).Return("https://example.com/art.jpg", nil)

			// Mock HTTP request for Discord external assets API
			assetsReq := &pdk.HTTPRequest{}
			pdk.PDKMock.On("NewHTTPRequest", pdk.MethodPost, mock.MatchedBy(func(url string) bool {
				return strings.Contains(url, "external-assets")
			})).Return(assetsReq)
			pdk.PDKMock.On("Send", assetsReq).Return(pdk.NewStubHTTPResponse(200, nil, []byte(`{"key":"test-key"}`)))

			// Schedule clear activity callback
			host.SchedulerMock.On("ScheduleOneTime", mock.Anything, payloadClearActivity, "testuser-clear").Return("testuser-clear", nil)

			err := plugin.NowPlaying(scrobbler.NowPlayingRequest{
				Username: "testuser",
				Position: 10,
				Track: scrobbler.TrackInfo{
					ID:       "track1",
					Title:    "Test Song",
					Artist:   "Test Artist",
					Album:    "Test Album",
					Duration: 180,
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Scrobble", func() {
		It("does nothing (returns nil)", func() {
			err := plugin.Scrobble(scrobbler.ScrobbleRequest{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("OnCallback", func() {
		BeforeEach(func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
		})

		It("handles heartbeat callback", func() {
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(42), true, nil)
			host.WebSocketMock.On("SendText", "testuser", mock.Anything).Return(nil)

			err := plugin.OnCallback(scheduler.SchedulerCallbackRequest{
				ScheduleID:  "testuser",
				Payload:     payloadHeartbeat,
				IsRecurring: true,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("handles clearActivity callback", func() {
			host.WebSocketMock.On("SendText", "testuser", mock.Anything).Return(nil)
			host.SchedulerMock.On("CancelSchedule", "testuser").Return(nil)
			host.WebSocketMock.On("CloseConnection", "testuser", int32(1000), "Navidrome disconnect").Return(nil)

			err := plugin.OnCallback(scheduler.SchedulerCallbackRequest{
				ScheduleID: "testuser-clear",
				Payload:    payloadClearActivity,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("logs warning for unknown payload", func() {
			err := plugin.OnCallback(scheduler.SchedulerCallbackRequest{
				ScheduleID: "testuser",
				Payload:    "unknown",
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
