package server

import (
	"io/fs"
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

var _ = Describe("createUnixSocketFile", func() {
	var socketPath string

	BeforeEach(func() {
		tempDir, _ := os.MkdirTemp("", "create_unix_socket_file_test")
		socketPath = filepath.Join(tempDir, "test.sock")
		DeferCleanup(func() {
			_ = os.RemoveAll(tempDir)
		})
	})

	When("unixSocketPerm is valid", func() {
		It("updates the permission of the unix socket file and returns nil", func() {
			_, err := createUnixSocketFile(socketPath, "0777")
			fileInfo, _ := os.Stat(socketPath)
			actualPermission := fileInfo.Mode().Perm()

			Expect(actualPermission).To(Equal(os.FileMode(0777)))
			Expect(err).ToNot(HaveOccurred())

		})
	})

	When("unixSocketPerm is invalid", func() {
		It("returns an error", func() {
			_, err := createUnixSocketFile(socketPath, "invalid")
			Expect(err).To(HaveOccurred())

		})
	})

	When("file already exists", func() {
		It("recreates the file as a socket with the right permissions", func() {
			_, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.Chmod(socketPath, os.FileMode(0777))).To(Succeed())

			_, err = createUnixSocketFile(socketPath, "0600")
			Expect(err).ToNot(HaveOccurred())
			fileInfo, _ := os.Stat(socketPath)
			Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0600)))
			Expect(fileInfo.Mode().Type()).To(Equal(fs.ModeSocket))
		})
	})
})
