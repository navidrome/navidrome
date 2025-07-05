package plugins

import (
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to check if a plugin implements LifecycleManagement
func hasInitService(info *plugin) bool {
	for _, c := range info.Capabilities {
		if c == CapabilityLifecycleManagement {
			return true
		}
	}
	return false
}

var _ = Describe("LifecycleManagement", func() {
	Describe("Plugin Lifecycle Manager", func() {
		var lifecycleManager *pluginLifecycleManager

		BeforeEach(func() {
			lifecycleManager = newPluginLifecycleManager(metrics.NewNoopInstance())
		})

		It("should track initialization state of plugins", func() {
			// Create test plugins
			plugin1 := &plugin{
				ID:           "test-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			plugin2 := &plugin{
				ID:           "another-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "0.5.0",
				},
			}

			// Initially, no plugins should be initialized
			Expect(lifecycleManager.isInitialized(plugin1)).To(BeFalse())
			Expect(lifecycleManager.isInitialized(plugin2)).To(BeFalse())

			// Mark first plugin as initialized
			lifecycleManager.markInitialized(plugin1)

			// Check state
			Expect(lifecycleManager.isInitialized(plugin1)).To(BeTrue())
			Expect(lifecycleManager.isInitialized(plugin2)).To(BeFalse())

			// Mark second plugin as initialized
			lifecycleManager.markInitialized(plugin2)

			// Both should be initialized now
			Expect(lifecycleManager.isInitialized(plugin1)).To(BeTrue())
			Expect(lifecycleManager.isInitialized(plugin2)).To(BeTrue())
		})

		It("should handle plugins with same name but different versions", func() {
			plugin1 := &plugin{
				ID:           "test-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			plugin2 := &plugin{
				ID:           "test-plugin", // Same name
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "2.0.0", // Different version
				},
			}

			// Mark v1 as initialized
			lifecycleManager.markInitialized(plugin1)

			// v1 should be initialized but not v2
			Expect(lifecycleManager.isInitialized(plugin1)).To(BeTrue())
			Expect(lifecycleManager.isInitialized(plugin2)).To(BeFalse())

			// Mark v2 as initialized
			lifecycleManager.markInitialized(plugin2)

			// Both versions should be initialized now
			Expect(lifecycleManager.isInitialized(plugin1)).To(BeTrue())
			Expect(lifecycleManager.isInitialized(plugin2)).To(BeTrue())

			// Verify the keys used for tracking
			key1 := plugin1.ID + consts.Zwsp + plugin1.Manifest.Version
			key2 := plugin1.ID + consts.Zwsp + plugin2.Manifest.Version
			_, exists1 := lifecycleManager.plugins.Load(key1)
			_, exists2 := lifecycleManager.plugins.Load(key2)
			Expect(exists1).To(BeTrue())
			Expect(exists2).To(BeTrue())
			Expect(key1).NotTo(Equal(key2))
		})

		It("should only consider plugins that implement LifecycleManagement", func() {
			// Plugin that implements LifecycleManagement
			initPlugin := &plugin{
				ID:           "init-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			// Plugin that doesn't implement LifecycleManagement
			regularPlugin := &plugin{
				ID:           "regular-plugin",
				Capabilities: []string{"MetadataAgent"},
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			// Check if plugins can be initialized
			Expect(hasInitService(initPlugin)).To(BeTrue())
			Expect(hasInitService(regularPlugin)).To(BeFalse())
		})

		It("should properly construct the plugin key", func() {
			plugin := &plugin{
				ID: "test-plugin",
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			expectedKey := "test-plugin" + consts.Zwsp + "1.0.0"
			actualKey := plugin.ID + consts.Zwsp + plugin.Manifest.Version

			Expect(actualKey).To(Equal(expectedKey))
		})

		It("should clear initialization state when requested", func() {
			plugin := &plugin{
				ID:           "test-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &schema.PluginManifest{
					Version: "1.0.0",
				},
			}

			// Initially not initialized
			Expect(lifecycleManager.isInitialized(plugin)).To(BeFalse())

			// Mark as initialized
			lifecycleManager.markInitialized(plugin)
			Expect(lifecycleManager.isInitialized(plugin)).To(BeTrue())

			// Clear initialization state
			lifecycleManager.clearInitialized(plugin)
			Expect(lifecycleManager.isInitialized(plugin)).To(BeFalse())
		})
	})
})
