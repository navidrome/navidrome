package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var samplePlugins = model.Plugins{
	{ID: "alpha", Manifest: `{"name":"Alpha","version":"1.0.0","author":"me"}`, Enabled: true},
	{ID: "beta", Manifest: `{"name":"Beta","version":"2.1.0","author":"me"}`, Enabled: false, LastError: "boom"},
}

var _ = Describe("plugin command format flags", func() {
	// Regression: list and info must not share a format variable. Binding the
	// same var on both commands makes the last init() registration clobber the
	// other's default, breaking `plugin list` with no -f flag.
	It("defaults `list -f` to table", func() {
		Expect(pluginListCmd.Flags().Lookup("format").DefValue).To(Equal("table"))
	})

	It("defaults `info -f` to text", func() {
		Expect(pluginInfoCmd.Flags().Lookup("format").DefValue).To(Equal("text"))
	})
})

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
	var cur *model.Plugin
	BeforeEach(func() {
		cur = &model.Plugin{ID: "alpha", Users: `["bob"]`, Libraries: `[1,2]`, AllowWriteAccess: true}
	})

	It("validates then updates config when config is provided", func() {
		mgr := &tests.MockPluginManager{}
		cfg := `{"key":"val"}`
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{config: &cfg})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.ValidatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.ValidatePluginConfigCalls[0].ConfigJSON).To(Equal(cfg))
		Expect(mgr.UpdatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginConfigCalls[0].ConfigJSON).To(Equal(cfg))
	})

	It("updates users with allUsers flag", func() {
		mgr := &tests.MockPluginManager{}
		all := true
		err := applyPluginEdit(context.Background(), mgr, cur,
			pluginEditOptions{allUsers: &all})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginUsersCalls[0].AllUsers).To(BeTrue())
	})

	It("updates libraries with allLibraries and write access", func() {
		mgr := &tests.MockPluginManager{}
		all := true
		wr := true
		err := applyPluginEdit(context.Background(), mgr, cur,
			pluginEditOptions{allLibraries: &all, writeAccess: &wr})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllLibraries).To(BeTrue())
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllowWriteAccess).To(BeTrue())
	})

	It("preserves existing fields when only the write-access flag changes", func() {
		mgr := &tests.MockPluginManager{}
		no := false
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{writeAccess: &no})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginLibrariesCalls[0].LibrariesJSON).To(Equal(`[1,2]`)) // not wiped
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllowWriteAccess).To(BeFalse())
	})

	It("preserves existing users when only the all-users flag changes", func() {
		mgr := &tests.MockPluginManager{}
		all := true
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{allUsers: &all})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls[0].UsersJSON).To(Equal(`["bob"]`)) // not wiped
	})

	It("parses a comma-separated users value into a JSON array", func() {
		mgr := &tests.MockPluginManager{}
		users := "alice, bob"
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{users: &users})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls[0].UsersJSON).To(Equal(`["alice","bob"]`))
	})

	It("passes a JSON-array users value through unchanged", func() {
		mgr := &tests.MockPluginManager{}
		users := `["alice","bob"]`
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{users: &users})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls[0].UsersJSON).To(Equal(`["alice","bob"]`))
	})

	It("rejects a malformed JSON-array users value", func() {
		mgr := &tests.MockPluginManager{}
		users := `["alice"`
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{users: &users})
		Expect(err).To(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls).To(BeEmpty())
	})

	It("parses a comma-separated libraries value into a JSON array of ints", func() {
		mgr := &tests.MockPluginManager{}
		libs := "1, 2"
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{libraries: &libs})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls[0].LibrariesJSON).To(Equal(`[1,2]`))
	})

	It("rejects a non-integer library ID", func() {
		mgr := &tests.MockPluginManager{}
		libs := "1,abc"
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{libraries: &libs})
		Expect(err).To(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls).To(BeEmpty())
	})

	It("clears allUsers when an explicit users list is set", func() {
		mgr := &tests.MockPluginManager{}
		cur.AllUsers = true
		users := "alice"
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{users: &users})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginUsersCalls[0].UsersJSON).To(Equal(`["alice"]`))
		Expect(mgr.UpdatePluginUsersCalls[0].AllUsers).To(BeFalse())
	})

	It("clears allLibraries when an explicit libraries list is set", func() {
		mgr := &tests.MockPluginManager{}
		cur.AllLibraries = true
		libs := "1"
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{libraries: &libs})
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.UpdatePluginLibrariesCalls[0].LibrariesJSON).To(Equal(`[1]`))
		Expect(mgr.UpdatePluginLibrariesCalls[0].AllLibraries).To(BeFalse())
	})

	It("does nothing and errors when no fields are set", func() {
		mgr := &tests.MockPluginManager{}
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{})
		Expect(err).To(HaveOccurred())
	})

	It("aborts the config update when validation fails", func() {
		mgr := &tests.MockPluginManager{ValidateError: errors.New("bad config")}
		cfg := `{"key":"val"}`
		err := applyPluginEdit(context.Background(), mgr, cur, pluginEditOptions{config: &cfg})
		Expect(err).To(HaveOccurred())
		Expect(mgr.ValidatePluginConfigCalls).To(HaveLen(1))
		Expect(mgr.UpdatePluginConfigCalls).To(BeEmpty())
	})
})

var _ = Describe("isPackagePath", func() {
	It("is true for any .ndp path", func() {
		Expect(isPackagePath("/some/dir/x.ndp")).To(BeTrue())
	})
	It("is true for a non-existent .ndp path (so ReadManifest reports the error)", func() {
		Expect(isPackagePath("/nope/x.ndp")).To(BeTrue())
	})
	It("is false for a bare plugin id", func() {
		Expect(isPackagePath("my-plugin")).To(BeFalse())
	})
})

var _ = Describe("formatPluginInfo", func() {
	It("renders installed plugin details as text", func() {
		p := &model.Plugin{ID: "alpha", Manifest: `{"name":"Alpha","version":"1.0.0","author":"me"}`, Enabled: true}
		out, err := formatPluginInfo(p, "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("alpha"))
		Expect(out).To(ContainSubstring("Alpha"))
	})
	It("renders json", func() {
		p := &model.Plugin{ID: "alpha", Manifest: `{}`}
		out, err := formatPluginInfo(p, "json")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("alpha"))
	})
})

var _ = Describe("formatManifestInfo", func() {
	It("renders text with name, version, author", func() {
		m := &plugins.Manifest{Name: "My Plugin", Version: "2.0.0", Author: "me"}
		out, err := formatManifestInfo(m, "abc123", "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("My Plugin"))
		Expect(out).To(ContainSubstring("2.0.0"))
		Expect(out).To(ContainSubstring("me"))
	})

	It("omits Description and Website when nil", func() {
		m := &plugins.Manifest{Name: "X", Version: "1.0.0", Author: "a"}
		out, err := formatManifestInfo(m, "abc123", "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).ToNot(ContainSubstring("Description:"))
		Expect(out).ToNot(ContainSubstring("Website:"))
	})

	It("includes Description and Website when set", func() {
		desc := "a cool plugin"
		site := "https://example.com"
		m := &plugins.Manifest{Name: "X", Version: "1.0.0", Author: "a", Description: &desc, Website: &site}
		out, err := formatManifestInfo(m, "abc123", "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("a cool plugin"))
		Expect(out).To(ContainSubstring("https://example.com"))
	})

	It("renders valid json", func() {
		m := &plugins.Manifest{Name: "X", Version: "1.0.0", Author: "a"}
		out, err := formatManifestInfo(m, "abc123", "json")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("\"name\""))
	})
})

var _ = Describe("formatPluginInfo enriched text", func() {
	var fixedTime time.Time

	BeforeEach(func() {
		fixedTime = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	})

	It("includes Users, Libraries, Permissions, Created, Updated in text output", func() {
		p := &model.Plugin{
			ID:        "myplugin",
			Manifest:  `{"name":"My Plugin","version":"1.0.0","author":"me","permissions":{"users":{},"subsonicapi":{}}}`,
			Enabled:   true,
			Users:     "alice,bob",
			Libraries: "1,2",
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime.Add(time.Hour),
		}
		out, err := formatPluginInfo(p, "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("alice,bob"))
		Expect(out).To(ContainSubstring("1,2"))
		Expect(out).To(ContainSubstring("subsonicapi"))
		Expect(out).To(ContainSubstring("users"))
		Expect(out).To(ContainSubstring("2025-01-15T12:00:00Z"))
	})

	It("omits Users and Libraries lines when empty", func() {
		p := &model.Plugin{
			ID:        "myplugin",
			Manifest:  `{"name":"X","version":"1.0.0","author":"me"}`,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}
		out, err := formatPluginInfo(p, "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).ToNot(ContainSubstring("Users:"))
		Expect(out).ToNot(ContainSubstring("Libraries:"))
	})

	It("does not alter json output", func() {
		p := &model.Plugin{ID: "x", Manifest: `{}`, Users: "alice"}
		out, err := formatPluginInfo(p, "json")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring(`"x"`))
	})
})

var _ = Describe("formatManifestInfo enriched text", func() {
	It("includes Permissions when declared", func() {
		p := &plugins.Permissions{Http: &plugins.HTTPPermission{}}
		m := &plugins.Manifest{Name: "P", Version: "1.0.0", Author: "a", Permissions: p}
		out, err := formatManifestInfo(m, "abc123", "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("http"))
		Expect(out).To(ContainSubstring("Permissions:"))
	})

	It("omits Permissions line when no permissions declared", func() {
		m := &plugins.Manifest{Name: "P", Version: "1.0.0", Author: "a"}
		out, err := formatManifestInfo(m, "abc123", "text")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).ToNot(ContainSubstring("Permissions:"))
	})

	It("does not alter json output", func() {
		p := &plugins.Permissions{Http: &plugins.HTTPPermission{}}
		m := &plugins.Manifest{Name: "P", Version: "1.0.0", Author: "a", Permissions: p}
		out, err := formatManifestInfo(m, "abc123", "json")
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring(`"name"`))
	})
})

var _ = Describe("rescanPlugins", func() {
	It("calls RescanPlugins on the manager", func() {
		mgr := &tests.MockPluginManager{}
		err := rescanPlugins(context.Background(), mgr)
		Expect(err).ToNot(HaveOccurred())
		Expect(mgr.RescanPluginsCalls).To(Equal(1))
	})
})
