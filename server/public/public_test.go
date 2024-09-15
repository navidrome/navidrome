package public

import (
	"net/http"
	"net/url"
	"path"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("publicURL", func() {
	When("ShareURL is set", func() {
		BeforeEach(func() {
			conf.Server.ShareURL = "http://share.myotherserver.com"
		})
		It("uses the config value instead of AbsoluteURL", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/share/123", nil)
			uri := path.Join(consts.URLPathPublic, "123")
			actual := publicURL(r, uri, nil)
			Expect(actual).To(Equal("http://share.myotherserver.com/share/123"))
		})
		It("concatenates params if provided", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/share/123", nil)
			uri := path.Join(consts.URLPathPublicImages, "123")
			params := url.Values{
				"size": []string{"300"},
			}
			actual := publicURL(r, uri, params)
			Expect(actual).To(Equal("http://share.myotherserver.com/share/img/123?size=300"))

		})
	})
	When("ShareURL is not set", func() {
		BeforeEach(func() {
			conf.Server.ShareURL = ""
		})
		It("uses AbsoluteURL", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/share/123", nil)
			uri := path.Join(consts.URLPathPublic, "123")
			actual := publicURL(r, uri, nil)
			Expect(actual).To(Equal("https://myserver.com/share/123"))
		})
		It("concatenates params if provided", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/share/123", nil)
			uri := path.Join(consts.URLPathPublicImages, "123")
			params := url.Values{
				"size": []string{"300"},
			}
			actual := publicURL(r, uri, params)
			Expect(actual).To(Equal("https://myserver.com/share/img/123?size=300"))
		})
	})
})
