package server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AbsoluteURL", func() {
	When("BaseURL is empty", func() {
		BeforeEach(func() {
			conf.Server.BasePath = ""
		})
		It("uses the scheme/host from the request", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
	When("BaseURL has only path", func() {
		BeforeEach(func() {
			conf.Server.BasePath = "/music"
		})
		It("uses the scheme/host from the request", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com/music/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
	When("BaseURL has full URL", func() {
		BeforeEach(func() {
			conf.Server.BaseScheme = "https"
			conf.Server.BaseHost = "myserver.com:8080"
			conf.Server.BasePath = "/music"
		})
		It("use the configured scheme/host/path", func() {
			r, _ := http.NewRequest("GET", "https://localhost:4533/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com:8080/music/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
})

var _ = Describe("updateSocketPermission", func() {
	tempDir, _ := os.MkdirTemp("", "")
	socketPath := filepath.Join(tempDir, "test.sock")
	file, _ := os.Create(socketPath)
	file.Close()

	When("unixSocketPerm is valid", func() {
		It("updates the permission of the unix socket file and returns nil", func() {
			err := updateSocketPermission(socketPath, "0017")
			fileInfo, _ := os.Stat(socketPath)
			actualPermission := fileInfo.Mode().Perm()

			Expect(actualPermission).To(Equal(os.FileMode(0017)))
			Expect(err).To(BeNil())
		})
	})

	When("unixSocketPerm is invalid", func() {
		It("returns an error", func() {
			err := updateSocketPermission(socketPath, "invalid")
			Expect(err).NotTo(BeNil())
		})

	})
})
