package nativeapi

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("maxImageUploadSize", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("returns the configured size when valid", func() {
		conf.Server.MaxImageUploadSize = "20MB"
		Expect(maxImageUploadSize()).To(Equal(int64(20_000_000)))
	})

	It("returns the default size when config is empty", func() {
		conf.Server.MaxImageUploadSize = ""
		Expect(maxImageUploadSize()).To(Equal(int64(10_000_000)))
	})

	It("returns the default size when config is invalid", func() {
		conf.Server.MaxImageUploadSize = "not-a-size"
		Expect(maxImageUploadSize()).To(Equal(int64(10_000_000)))
	})

	It("parses raw byte values", func() {
		conf.Server.MaxImageUploadSize = "52428800"
		Expect(maxImageUploadSize()).To(Equal(int64(52_428_800)))
	})
})
