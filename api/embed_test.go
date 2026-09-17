package api_test

import (
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/navidrome/navidrome/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Bundled spec", func() {
	It("embeds a valid OpenAPI 3 document", func() {
		doc, err := openapi3.NewLoader().LoadFromData(api.SpecJSON())
		Expect(err).ToNot(HaveOccurred())
		Expect(doc.Validate(GinkgoT().Context())).To(Succeed())
		Expect(doc.Paths.Find("/server")).ToNot(BeNil())
	})

	It("embeds the YAML variant", func() {
		var doc map[string]any
		Expect(yaml.Unmarshal(api.SpecYAML(), &doc)).To(Succeed())
		Expect(doc).To(HaveKey("paths"))
	})

	It("reports the version from the bundle, matching the source root document", func() {
		src, err := os.ReadFile("api/openapi/openapi.yaml")
		Expect(err).ToNot(HaveOccurred())
		var root struct {
			Info struct{ Version string } `yaml:"info"`
		}
		Expect(yaml.Unmarshal(src, &root)).To(Succeed())
		Expect(api.SpecVersion()).To(Equal(root.Info.Version))
		Expect(api.SpecVersion()).ToNot(BeEmpty())
	})
})
