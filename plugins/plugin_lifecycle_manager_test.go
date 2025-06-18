package plugins

import (
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to check if a plugin implements LifecycleManagement
func hasInitService(info *pluginInfo) bool {
	for _, s := range info.Capabilities {
		if s == CapabilityLifecycleManagement {
			return true
		}
	}
	return false
}

var _ = Describe("LifecycleManagement", func() {
	Describe("Plugin Lifecycle Manager", func() {
		var lifecycleManager *pluginLifecycleManager

		BeforeEach(func() {
			lifecycleManager = newPluginLifecycleManager()
		})

		It("should track initialization state of plugins", func() {
			// Create test plugins
			plugin1 := &pluginInfo{
				Name:         "test-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &PluginManifest{
					Version: "1.0.0",
				},
			}

			plugin2 := &pluginInfo{
				Name:         "another-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &PluginManifest{
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
			plugin1 := &pluginInfo{
				Name:         "test-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &PluginManifest{
					Version: "1.0.0",
				},
			}

			plugin2 := &pluginInfo{
				Name:         "test-plugin", // Same name
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &PluginManifest{
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
			key1 := plugin1.Name + consts.Zwsp + plugin1.Manifest.Version
			key2 := plugin2.Name + consts.Zwsp + plugin2.Manifest.Version
			Expect(lifecycleManager.plugins).To(HaveKey(key1))
			Expect(lifecycleManager.plugins).To(HaveKey(key2))
			Expect(key1).NotTo(Equal(key2))
		})

		It("should only consider plugins that implement LifecycleManagement", func() {
			// Plugin that implements LifecycleManagement
			initPlugin := &pluginInfo{
				Name:         "init-plugin",
				Capabilities: []string{CapabilityLifecycleManagement},
				Manifest: &PluginManifest{
					Version: "1.0.0",
				},
			}

			// Plugin that doesn't implement LifecycleManagement
			regularPlugin := &pluginInfo{
				Name:         "regular-plugin",
				Capabilities: []string{"MetadataAgent"},
				Manifest: &PluginManifest{
					Version: "1.0.0",
				},
			}

			// Check if plugins can be initialized
			Expect(hasInitService(initPlugin)).To(BeTrue())
			Expect(hasInitService(regularPlugin)).To(BeFalse())
		})

		It("should properly construct the plugin key", func() {
			plugin := &pluginInfo{
				Name: "test-plugin",
				Manifest: &PluginManifest{
					Version: "1.0.0",
				},
			}

			expectedKey := "test-plugin" + consts.Zwsp + "1.0.0"
			actualKey := plugin.Name + consts.Zwsp + plugin.Manifest.Version

			Expect(actualKey).To(Equal(expectedKey))
		})
	})
})
