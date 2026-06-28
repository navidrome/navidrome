package cmd

import (
	"encoding/json"
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var samplePlugins = model.Plugins{
	{ID: "alpha", Manifest: `{"name":"Alpha","version":"1.0.0","author":"me"}`, Enabled: true},
	{ID: "beta", Manifest: `{"name":"Beta","version":"2.1.0","author":"me"}`, Enabled: false, LastError: "boom"},
}

var _ = Describe("formatPluginList", func() {
	It("renders csv with a header and one row per plugin", func() {
		out, err := formatPluginList(samplePlugins, "csv")
		Expect(err).ToNot(HaveOccurred())
		lines := strings.Split(strings.TrimSpace(out), "\n")
		Expect(lines).To(HaveLen(3)) // header + 2 rows
		Expect(lines[0]).To(ContainSubstring("id"))
		Expect(out).To(ContainSubstring("alpha"))
		Expect(out).To(ContainSubstring("beta"))
	})

	It("renders valid json", func() {
		out, err := formatPluginList(samplePlugins, "json")
		Expect(err).ToNot(HaveOccurred())
		var got []map[string]any
		Expect(json.Unmarshal([]byte(out), &got)).To(Succeed())
		Expect(got).To(HaveLen(2))
	})

	It("renders a human table by default", func() {
		out, err := formatPluginList(samplePlugins, "table")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("Alpha"))
		Expect(out).To(ContainSubstring("1.0.0"))
	})

	It("errors on an unknown format", func() {
		_, err := formatPluginList(samplePlugins, "yaml")
		Expect(err).To(HaveOccurred())
	})
})
