package metrics

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("installedPackage", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
	})

	It("returns empty when there's no .package file", func() {
		Expect(installedPackage()).To(BeEmpty())
	})

	It("reads the .package file from the data folder", func() {
		writePackageFile("deb")

		Expect(installedPackage()).To(Equal("deb"))
	})

	It("trims surrounding whitespace, as the msi packager writes a trailing newline", func() {
		writePackageFile("msi\n")

		Expect(installedPackage()).To(Equal("msi"))
	})

	It("ignores ND_PLATFORM", func() {
		GinkgoT().Setenv("ND_PLATFORM", "zimaos")

		Expect(installedPackage()).To(BeEmpty())
	})
})

var _ = Describe("hostingPlatform", func() {
	It("returns empty when ND_PLATFORM is not set", func() {
		Expect(hostingPlatform()).To(BeEmpty())
	})

	It("reads ND_PLATFORM", func() {
		GinkgoT().Setenv("ND_PLATFORM", "zimaos")

		Expect(hostingPlatform()).To(Equal("zimaos"))
	})

	It("trims surrounding whitespace", func() {
		GinkgoT().Setenv("ND_PLATFORM", "  pikapods\n")

		Expect(hostingPlatform()).To(Equal("pikapods"))
	})
})

func writePackageFile(content string) {
	GinkgoHelper()
	path := filepath.Join(conf.Server.DataFolder.String(), ".package")
	Expect(os.WriteFile(path, []byte(content), 0600)).To(Succeed())
}
