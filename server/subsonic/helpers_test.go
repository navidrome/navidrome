package subsonic

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subsonic Helpers", func() {
	Describe("getBaseArtistImageUrl", func() {
		var r *http.Request
		var params url.Values

		BeforeEach(func() {
			params = url.Values{}
			params.Add("c", "TestUI")
			params.Add("s", "abcdef")
			params.Add("t", "12345")
			params.Add("u", "test")
			params.Add("v", "1.8.test")
			r = httptest.NewRequest("GET", fmt.Sprintf("/arbitraryUrl?%s", params.Encode()), nil)
		})

		AfterEach(func() {
			conf.Server.BaseDomain = ""
		})

		It("should return empty string if no domain provided", func() {
			resp := getBaseArtistImageUrl(r)
			Expect(resp).To(Equal(""))
		})

		It("should return meaningful string if domain provided", func() {
			conf.Server.BaseDomain = "https://example.com"
			resp := getBaseArtistImageUrl(r)
			Expect(strings.HasPrefix(resp, "https://example.com/rest/getCoverArt?")).To(BeTrue())

			query, err := url.ParseQuery(resp)

			Expect(err).To(BeNil())
			Expect(query.Get("c")).To(Equal(params.Get("c")))
			Expect(query.Get("f")).To(Equal("json"))
			Expect(query.Get("s")).To(Equal(params.Get("s")))
			Expect(query.Get("t")).To(Equal(params.Get("t")))
			Expect(query.Get("u")).To(Equal(params.Get("u")))
			Expect(query.Get("v")).To(Equal(params.Get("v")))

			Expect(strings.Contains(resp, "_="))
		})

		It("should return meaningful string if domain and base url provided", func() {
			conf.Server.BaseDomain = "https://example.com"
			conf.Server.BaseURL = "/subroute"
			resp := getBaseArtistImageUrl(r)
			Expect(strings.HasPrefix(resp, "https://example.com/subroute/rest/getCoverArt?")).To(BeTrue())

			query, err := url.ParseQuery(resp)

			Expect(err).To(BeNil())
			Expect(query.Get("c")).To(Equal(params.Get("c")))
			Expect(query.Get("f")).To(Equal("json"))
			Expect(query.Get("s")).To(Equal(params.Get("s")))
			Expect(query.Get("t")).To(Equal(params.Get("t")))
			Expect(query.Get("u")).To(Equal(params.Get("u")))
			Expect(query.Get("v")).To(Equal(params.Get("v")))

			Expect(strings.Contains(resp, "_="))
		})
	})
})
