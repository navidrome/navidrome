package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
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

var _ = Describe("enable/disable plugin", func() {
	It("calls EnablePlugin on the manager", func() {
		mgr := &tests.MockPluginManager{}
		err := enablePlugin(context.Background(), mgr, "alpha")
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.EnablePluginCalls).To(Equal([]string{"alpha"}))
	})

	It("calls DisablePlugin on the manager", func() {
		mgr := &tests.MockPluginManager{}
		err := disablePlugin(context.Background(), mgr, "beta")
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.DisablePluginCalls).To(Equal([]string{"beta"}))
	})
})

var _ = Describe("applyPluginEdit", func() {
	It("validates then updates config when config is provided", func() {
		mgr := &tests.MockPluginManager{}
		cfg := `{"key":"val"}`
		err := applyPluginEdit(context.Background(), mgr, "alpha", pluginEditOptions{config: &cfg})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.ValidatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.ValidatePluginConfigCalls[0].ConfigJSON).To(Equal(cfg))
		Expect(mgr.UpdatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginConfigCalls[0].ConfigJSON).To(Equal(cfg))
	})

	It("updates users with allUsers flag", func() {
		mgr := &tests.MockPluginManager{}
		all := true
		err := applyPluginEdit(context.Background(), mgr, "alpha",
			pluginEditOptions{allUsers: &all})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginUsersCalls[0].AllUsers).To(BeTrue())
	})

	It("updates libraries with allLibraries and write access", func() {
		mgr := &tests.MockPluginManager{}
		all := true
		wr := true
		err := applyPluginEdit(context.Background(), mgr, "alpha",
			pluginEditOptions{allLibraries: &all, writeAccess: &wr})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllLibraries).To(BeTrue())
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllowWriteAccess).To(BeTrue())
	})

	It("does nothing and errors when no fields are set", func() {
		mgr := &tests.MockPluginManager{}
		err := applyPluginEdit(context.Background(), mgr, "alpha", pluginEditOptions{})
		Expect(err).To(HaveOccurred())
	})

	It("aborts the config update when validation fails", func() {
		mgr := &tests.MockPluginManager{ValidateError: errors.New("bad config")}
		cfg := `{"key":"val"}`
		err := applyPluginEdit(context.Background(), mgr, "alpha", pluginEditOptions{config: &cfg})
		Expect(err).To(HaveOccurred())
		Expect(mgr.ValidatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginConfigCalls).To(BeEmpty())
	})
})
