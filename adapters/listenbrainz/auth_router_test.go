package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ListenBrainz Auth Router", func() {
	var sk *fakeSessionKeys
	var httpClient *tests.FakeHttpClient
	var r Router
	var req *http.Request
	var resp *httptest.ResponseRecorder

	BeforeEach(func() {
		sk = &fakeSessionKeys{KeyName: sessionKeyProperty}
		httpClient = &tests.FakeHttpClient{}
		cl := newClient("http://localhost/", httpClient)
		r = Router{
			sessionKeys: sk,
			client:      cl,
		}
		resp = httptest.NewRecorder()
	})

	Describe("getLinkStatus", func() {
		It("returns false when there is no stored session key", func() {
			req = httptest.NewRequest("GET", "/listenbrainz/link", nil)
			r.getLinkStatus(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			var parsed map[string]interface{}
			Expect(json.Unmarshal(resp.Body.Bytes(), &parsed)).To(BeNil())
			Expect(parsed["status"]).To(Equal(false))
		})

		It("returns true when there is a stored session key", func() {
			sk.KeyValue = "sk-1"
			req = httptest.NewRequest("GET", "/listenbrainz/link", nil)
			r.getLinkStatus(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			var parsed map[string]interface{}
			Expect(json.Unmarshal(resp.Body.Bytes(), &parsed)).To(BeNil())
			Expect(parsed["status"]).To(Equal(true))
		})
	})

	Describe("link", func() {
		It("returns bad request when no token is sent", func() {
			req = httptest.NewRequest("PUT", "/listenbrainz/link", strings.NewReader(`{}`))
			r.link(resp, req)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns bad request when the token is invalid", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 200, "message": "Token invalid.", "valid": false}`)),
				StatusCode: 200,
			}

			req = httptest.NewRequest("PUT", "/listenbrainz/link", strings.NewReader(`{"token": "invalid-tok-1"}`))
			r.link(resp, req)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns true and the username when the token is valid", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 200, "message": "Token valid.", "user_name": "ListenBrainzUser", "valid": true}`)),
				StatusCode: 200,
			}

			req = httptest.NewRequest("PUT", "/listenbrainz/link", strings.NewReader(`{"token": "tok-1"}`))
			r.link(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			var parsed map[string]interface{}
			Expect(json.Unmarshal(resp.Body.Bytes(), &parsed)).To(BeNil())
			Expect(parsed["status"]).To(Equal(true))
			Expect(parsed["user"]).To(Equal("ListenBrainzUser"))
		})

		It("saves the session key when the token is valid", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 200, "message": "Token valid.", "user_name": "ListenBrainzUser", "valid": true}`)),
				StatusCode: 200,
			}

			req = httptest.NewRequest("PUT", "/listenbrainz/link", strings.NewReader(`{"token": "tok-1"}`))
			r.link(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(sk.KeyValue).To(Equal("tok-1"))
		})
	})

	Describe("unlink", func() {
		It("removes the session key when unlinking", func() {
			sk.KeyValue = "tok-1"
			req = httptest.NewRequest("DELETE", "/listenbrainz/link", nil)
			r.unlink(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(sk.KeyValue).To(Equal(""))
		})
	})
})

type fakeSessionKeys struct {
	KeyName  string
	KeyValue string
}

func (sk *fakeSessionKeys) Put(ctx context.Context, userId, sessionKey string) error {
	sk.KeyValue = sessionKey
	return nil
}

func (sk *fakeSessionKeys) Get(ctx context.Context, userId string) (string, error) {
	return sk.KeyValue, nil
}

func (sk *fakeSessionKeys) Delete(ctx context.Context, userId string) error {
	sk.KeyValue = ""
	return nil
}
