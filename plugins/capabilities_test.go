package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockFunctionChecker implements functionExistsChecker for testing
type mockFunctionChecker struct {
	functions map[string]bool
}

func (m *mockFunctionChecker) FunctionExists(name string) bool {
	return m.functions[name]
}

var _ = Describe("Capabilities", func() {
	Describe("detectCapabilities", func() {
		It("detects MetadataAgent capability when plugin exports artist biography function", func() {
			checker := &mockFunctionChecker{
				functions: map[string]bool{
					FuncGetArtistBiography: true,
				},
			}

			caps := detectCapabilities(checker)
			Expect(caps).To(ContainElement(CapabilityMetadataAgent))
		})

		It("detects MetadataAgent capability when plugin exports multiple functions", func() {
			checker := &mockFunctionChecker{
				functions: map[string]bool{
					FuncGetArtistMBID:  true,
					FuncGetArtistURL:   true,
					FuncGetAlbumInfo:   true,
					FuncGetAlbumImages: true,
				},
			}

			caps := detectCapabilities(checker)
			Expect(caps).To(ContainElement(CapabilityMetadataAgent))
			Expect(caps).To(HaveLen(1)) // Should only have one MetadataAgent capability
		})

		It("returns empty slice when no capability functions are exported", func() {
			checker := &mockFunctionChecker{
				functions: map[string]bool{
					"some_other_function": true,
				},
			}

			caps := detectCapabilities(checker)
			Expect(caps).To(BeEmpty())
		})

		It("returns empty slice when plugin exports no functions", func() {
			checker := &mockFunctionChecker{
				functions: map[string]bool{},
			}

			caps := detectCapabilities(checker)
			Expect(caps).To(BeEmpty())
		})
	})

	Describe("hasCapability", func() {
		It("returns true when capability exists", func() {
			caps := []Capability{CapabilityMetadataAgent}
			Expect(hasCapability(caps, CapabilityMetadataAgent)).To(BeTrue())
		})

		It("returns false when capability does not exist", func() {
			var caps []Capability
			Expect(hasCapability(caps, CapabilityMetadataAgent)).To(BeFalse())
		})

		It("returns false when capabilities slice is nil", func() {
			Expect(hasCapability(nil, CapabilityMetadataAgent)).To(BeFalse())
		})
	})
})
