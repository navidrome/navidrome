package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("System Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("ping", func() {
		It("returns a successful response", func() {
			r := newReq("ping")
			resp, err := router.Ping(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})
	})

	Describe("getLicense", func() {
		It("returns a valid license", func() {
			r := newReq("getLicense")
			resp, err := router.GetLicense(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.License).ToNot(BeNil())
			Expect(resp.License.Valid).To(BeTrue())
		})
	})

	Describe("getOpenSubsonicExtensions", func() {
		It("returns a list of supported extensions", func() {
			r := newReq("getOpenSubsonicExtensions")
			resp, err := router.GetOpenSubsonicExtensions(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.OpenSubsonicExtensions).ToNot(BeNil())
			Expect(*resp.OpenSubsonicExtensions).ToNot(BeEmpty())
		})

		It("includes the transcodeOffset extension", func() {
			r := newReq("getOpenSubsonicExtensions")
			resp, err := router.GetOpenSubsonicExtensions(r)

			Expect(err).ToNot(HaveOccurred())
			extensions := *resp.OpenSubsonicExtensions
			var names []string
			for _, ext := range extensions {
				names = append(names, ext.Name)
			}
			Expect(names).To(ContainElement("transcodeOffset"))
		})

		It("includes the formPost extension", func() {
			r := newReq("getOpenSubsonicExtensions")
			resp, err := router.GetOpenSubsonicExtensions(r)

			Expect(err).ToNot(HaveOccurred())
			extensions := *resp.OpenSubsonicExtensions
			var names []string
			for _, ext := range extensions {
				names = append(names, ext.Name)
			}
			Expect(names).To(ContainElement("formPost"))
		})

		It("includes the songLyrics extension", func() {
			r := newReq("getOpenSubsonicExtensions")
			resp, err := router.GetOpenSubsonicExtensions(r)

			Expect(err).ToNot(HaveOccurred())
			extensions := *resp.OpenSubsonicExtensions
			var names []string
			for _, ext := range extensions {
				names = append(names, ext.Name)
			}
			Expect(names).To(ContainElement("songLyrics"))
		})
	})
})
