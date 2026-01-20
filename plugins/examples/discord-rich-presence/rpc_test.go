package main

import (
	"errors"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/websocket"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("discordRPC", func() {
	var r *discordRPC

	BeforeEach(func() {
		r = &discordRPC{}
		pdk.ResetMock()
		host.CacheMock.ExpectedCalls = nil
		host.CacheMock.Calls = nil
		host.WebSocketMock.ExpectedCalls = nil
		host.WebSocketMock.Calls = nil
		host.SchedulerMock.ExpectedCalls = nil
		host.SchedulerMock.Calls = nil
	})

	Describe("sendMessage", func() {
		It("sends JSON message over WebSocket", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":3`)
			})).Return(nil)

			err := r.sendMessage("testuser", presenceOpCode, map[string]string{"status": "online"})
			Expect(err).ToNot(HaveOccurred())
			host.WebSocketMock.AssertExpectations(GinkgoT())
		})

		It("returns error when WebSocket send fails", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.WebSocketMock.On("SendText", mock.Anything, mock.Anything).
				Return(errors.New("connection closed"))

			err := r.sendMessage("testuser", presenceOpCode, map[string]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection closed"))
		})
	})

	Describe("sendHeartbeat", func() {
		It("retrieves sequence number from cache and sends heartbeat", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(123), true, nil)
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":1`) && strings.Contains(msg, "123")
			})).Return(nil)

			err := r.sendHeartbeat("testuser")
			Expect(err).ToNot(HaveOccurred())
			host.CacheMock.AssertExpectations(GinkgoT())
			host.WebSocketMock.AssertExpectations(GinkgoT())
		})

		It("returns error when cache get fails", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(0), false, errors.New("cache error"))

			err := r.sendHeartbeat("testuser")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cache error"))
		})
	})

	Describe("connect", func() {
		It("establishes WebSocket connection and sends identify payload", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(0), false, errors.New("not found"))

			// Mock HTTP GET request for gateway discovery
			gatewayResp := []byte(`{"url":"wss://gateway.discord.gg"}`)
			httpReq := &pdk.HTTPRequest{}
			pdk.PDKMock.On("NewHTTPRequest", pdk.MethodGet, "https://discord.com/api/gateway").Return(httpReq)
			pdk.PDKMock.On("Send", mock.Anything).Return(pdk.NewStubHTTPResponse(200, nil, gatewayResp))

			// Mock WebSocket connection
			host.WebSocketMock.On("Connect", mock.MatchedBy(func(url string) bool {
				return strings.Contains(url, "gateway.discord.gg")
			}), mock.Anything, "testuser").Return("testuser", nil)
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":2`) && strings.Contains(msg, "test-token")
			})).Return(nil)
			host.SchedulerMock.On("ScheduleRecurring", "@every 41s", payloadHeartbeat, "testuser").
				Return("testuser", nil)

			err := r.connect("testuser", "test-token")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reuses existing connection if connected", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(42), true, nil)
			host.WebSocketMock.On("SendText", "testuser", mock.Anything).Return(nil)

			err := r.connect("testuser", "test-token")
			Expect(err).ToNot(HaveOccurred())
			host.WebSocketMock.AssertNotCalled(GinkgoT(), "Connect", mock.Anything, mock.Anything, mock.Anything)
		})
	})

	Describe("disconnect", func() {
		It("cancels schedule and closes WebSocket connection", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.SchedulerMock.On("CancelSchedule", "testuser").Return(nil)
			host.WebSocketMock.On("CloseConnection", "testuser", int32(1000), "Navidrome disconnect").Return(nil)

			err := r.disconnect("testuser")
			Expect(err).ToNot(HaveOccurred())
			host.SchedulerMock.AssertExpectations(GinkgoT())
			host.WebSocketMock.AssertExpectations(GinkgoT())
		})
	})

	Describe("cleanupFailedConnection", func() {
		It("cancels schedule, closes WebSocket, and clears cache", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.SchedulerMock.On("CancelSchedule", "testuser").Return(nil)
			host.WebSocketMock.On("CloseConnection", "testuser", int32(1000), "Connection lost").Return(nil)
			host.CacheMock.On("Remove", "discord.seq.testuser").Return(nil)

			r.cleanupFailedConnection("testuser")

			host.SchedulerMock.AssertExpectations(GinkgoT())
			host.WebSocketMock.AssertExpectations(GinkgoT())
		})
	})

	Describe("handleHeartbeatCallback", func() {
		It("sends heartbeat successfully", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(42), true, nil)
			host.WebSocketMock.On("SendText", "testuser", mock.Anything).Return(nil)

			err := r.handleHeartbeatCallback("testuser")
			Expect(err).ToNot(HaveOccurred())
		})

		It("cleans up connection on heartbeat failure", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetInt", "discord.seq.testuser").Return(int64(0), false, errors.New("cache miss"))
			host.SchedulerMock.On("CancelSchedule", "testuser").Return(nil)
			host.WebSocketMock.On("CloseConnection", "testuser", int32(1000), "Connection lost").Return(nil)
			host.CacheMock.On("Remove", "discord.seq.testuser").Return(nil)

			err := r.handleHeartbeatCallback("testuser")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection cleaned up"))
		})
	})

	Describe("handleClearActivityCallback", func() {
		It("clears activity and disconnects", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":3`) && strings.Contains(msg, `"activities":null`)
			})).Return(nil)
			host.SchedulerMock.On("CancelSchedule", "testuser").Return(nil)
			host.WebSocketMock.On("CloseConnection", "testuser", int32(1000), "Navidrome disconnect").Return(nil)

			err := r.handleClearActivityCallback("testuser")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("WebSocket callbacks", func() {
		Describe("OnTextMessage", func() {
			It("handles valid JSON message", func() {
				pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
				host.CacheMock.On("SetInt", mock.Anything, mock.Anything, mock.Anything).Return(nil)

				err := r.OnTextMessage(websocket.OnTextMessageRequest{
					ConnectionID: "testuser",
					Message:      `{"s":42}`,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error for invalid JSON", func() {
				pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
				err := r.OnTextMessage(websocket.OnTextMessageRequest{
					ConnectionID: "testuser",
					Message:      `not json`,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("OnBinaryMessage", func() {
			It("handles binary message without error", func() {
				pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
				err := r.OnBinaryMessage(websocket.OnBinaryMessageRequest{
					ConnectionID: "testuser",
					Data:         "AQID", // base64 encoded [0x01, 0x02, 0x03]
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("OnError", func() {
			It("handles error without returning error", func() {
				pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
				err := r.OnError(websocket.OnErrorRequest{
					ConnectionID: "testuser",
					Error:        "test error",
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("OnClose", func() {
			It("handles close without returning error", func() {
				pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
				err := r.OnClose(websocket.OnCloseRequest{
					ConnectionID: "testuser",
					Code:         1000,
					Reason:       "normal close",
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("sendActivity", func() {
		BeforeEach(func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.CacheMock.On("GetString", mock.MatchedBy(func(key string) bool {
				return strings.HasPrefix(key, "discord.image.")
			})).Return("", false, nil)
			host.CacheMock.On("SetString", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock HTTP request for Discord external assets API (image processing)
			// When processImage is called, it makes an HTTP request
			httpReq := &pdk.HTTPRequest{}
			pdk.PDKMock.On("NewHTTPRequest", pdk.MethodPost, mock.Anything).Return(httpReq)
			pdk.PDKMock.On("Send", mock.Anything).Return(pdk.NewStubHTTPResponse(200, nil, []byte(`{"key":"test-key"}`)))
		})

		It("sends activity update to Discord", func() {
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":3`) &&
					strings.Contains(msg, `"name":"Test Song"`) &&
					strings.Contains(msg, `"state":"Test Artist"`)
			})).Return(nil)

			err := r.sendActivity("client123", "testuser", "token123", activity{
				Application: "client123",
				Name:        "Test Song",
				Type:        2,
				State:       "Test Artist",
				Details:     "Test Album",
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("clearActivity", func() {
		It("sends presence update with nil activities", func() {
			pdk.PDKMock.On("Log", mock.Anything, mock.Anything).Maybe()
			host.WebSocketMock.On("SendText", "testuser", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, `"op":3`) && strings.Contains(msg, `"activities":null`)
			})).Return(nil)

			err := r.clearActivity("testuser")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
