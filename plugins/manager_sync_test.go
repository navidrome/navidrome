package plugins

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComputeFileSHA256", func() {
	It("returns a consistent 64-char lowercase hex hash for the same file", func() {
		dir := GinkgoT().TempDir()
		ndpPath := filepath.Join(dir, "test.ndp")
		err := createTestPackage(ndpPath, &Manifest{Name: "S", Author: "a", Version: "1.0.0"}, []byte{0x00, 0x61, 0x73, 0x6d})
		Expect(err).ToNot(HaveOccurred())

		hash1, err := ComputeFileSHA256(ndpPath)
		Expect(err).ToNot(HaveOccurred())
		hash2, err := ComputeFileSHA256(ndpPath)
		Expect(err).ToNot(HaveOccurred())

		Expect(hash1).To(Equal(hash2))
		Expect(hash1).To(MatchRegexp(`^[0-9a-f]{64}$`))
	})

	It("returns an error for a non-existent path", func() {
		_, err := ComputeFileSHA256(filepath.Join(GinkgoT().TempDir(), "does-not-exist.ndp"))
		Expect(err).To(HaveOccurred())
	})
})
