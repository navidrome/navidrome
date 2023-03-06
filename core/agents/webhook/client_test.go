package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("client", func() {
	var httpClient *tests.FakeHttpClient
	var client *client

	BeforeEach(func() {
		httpClient = &tests.FakeHttpClient{}
		client = newClient("http://example.com:1000", "API_KEY", httpClient)
	})

	Describe("webhookResponse", func() {
		It("Parses an error properly", func() {
			var response webhookResponse
			err := json.Unmarshal([]byte(`{"error": 10, "message": "Something went wrong"}`), &response)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Error).To(Equal(10))
			Expect(response.Message).To(Equal("Something went wrong"))
			Expect(response.UserName).To(BeEmpty())
		})

		It("Parses a validation properly", func() {
			var response webhookResponse
			err := json.Unmarshal([]byte(`{"userName": "me"}`), &response)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Error).To(Equal(0))
			Expect(response.Message).To(BeEmpty())
			Expect(response.UserName).To(Equal("me"))
		})
	})

	Describe("Bad requests", func() {
		testReq := func(test, body string, code int, targetErr error) {
			It(test, func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					StatusCode: code,
				}

				_, err := client.makeRequest(context.Background(), "/endpoint", map[string]interface{}{})

				if targetErr == nil {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).To(Equal(targetErr))

				}
			})
		}

		testReq("Bad Status Code", "{}", 400, fmt.Errorf("webhook http status: (400)"))
		testReq("Malformed json", "{", 200, nil)
		testReq("Error message", `{ "error": 5, "message": "No" }`, 200, &webhookError{
			Code:    5,
			Message: "No",
		})
	})

	Describe("validateToken", func() {
		BeforeEach(func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"userName":"example"}`)),
				StatusCode: 200,
			}
		})

		It("Formats a request properly", func() {
			_, err := client.validateToken(context.Background(), "WB-TOKEN")
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal("http://example.com:1000/validate"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

			decoder := json.NewDecoder(httpClient.SavedRequest.Body)
			var requestBody map[string]interface{}

			err = decoder.Decode(&requestBody)
			Expect(err).ToNot(HaveOccurred())

			Expect(requestBody).To(Equal(map[string]interface{}{
				"apiKey": "API_KEY",
				"token":  "WB-TOKEN",
			}))
		})

		It("passes and returns the response", func() {
			res, err := client.validateToken(context.Background(), "WB-TOKEN")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Error).To(Equal(0))
			Expect(res.UserName).To(Equal("example"))
		})
	})

	Describe("scrobble", func() {
		var info ScrobbleInfo
		BeforeEach(func() {
			info = ScrobbleInfo{
				artist:      "Track Artist",
				track:       "Track Title",
				album:       "Track Album",
				albumArtist: "Album Artist",
				trackNumber: 5,
				mbid:        "mbz-356",
				duration:    410,
			}
		})

		Describe("sending the request", func() {
			BeforeEach(func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					StatusCode: 200,
				}
			})

			test := func(isSubmission bool) {
				b := strconv.FormatBool(isSubmission)
				It("Formats request when submission = "+b, func() {
					if isSubmission {
						info.timestamp = time.Unix(0, 0)
					}
					Expect(client.scrobble(context.Background(), "token", isSubmission, info)).To(Succeed())
					Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
					Expect(httpClient.SavedRequest.URL.String()).To(Equal("http://example.com:1000/scrobble"))
					Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

					body, _ := io.ReadAll(httpClient.SavedRequest.Body)
					f, _ := os.ReadFile("tests/fixtures/webhook.scrobble." + b + ".request.json")
					Expect(body).To(MatchJSON(f))
				})
			}

			test(false)
			test(true)
		})
	})
})
