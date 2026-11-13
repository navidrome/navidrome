package utils_test

import (
	"time"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RelPath on Windows", func() {
	type result struct {
		rel string
		err error
	}

	DescribeTable("handles UNC share roots without hanging",
		func(base, target, expected string) {
			done := make(chan result, 1)
			go func() {
				rel, err := utils.RelPath(base, target)
				done <- result{rel, err}
			}()
			var res result
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive(&res))
			Expect(res.err).ToNot(HaveOccurred())
			Expect(res.rel).To(Equal(expected))
		},
		Entry("root and root with trailing separator", `\\server\Music`, `\\server\Music\`, "."),
		Entry("root with trailing separator and root", `\\server\Music\`, `\\server\Music`, "."),
		Entry("root and root with many separators", `\\server\Music`, `\\server\Music\\`, "."),
		Entry("root with forward slashes", `//server/Music`, `//server/Music/`, "."),
		Entry("root and child folder", `\\server\Music`, `\\server\Music\Artist`, "Artist"),
		Entry("root with trailing separator and child folder", `\\server\Music\`, `\\server\Music\Artist\Album`, `Artist\Album`),
		Entry("child folder and root", `\\server\Music\Artist`, `\\server\Music\`, ".."),
		Entry("child folder and root without trailing separator", `\\server\Music\Artist`, `\\server\Music`, ".."),
		Entry("root and root with different case", `\\server\Music`, `\\SERVER\music\`, "."),
		Entry("drive root and child folder", `C:\`, `C:\Music`, "Music"),
	)
})
