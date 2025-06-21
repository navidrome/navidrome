package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	hostconfig "github.com/navidrome/navidrome/plugins/host/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("configServiceImpl", func() {
	var (
		svc        *configServiceImpl
		pluginName string
	)

	BeforeEach(func() {
		pluginName = "testplugin"
		svc = &configServiceImpl{pluginID: pluginName}
		conf.Server.PluginConfig = map[string]map[string]string{
			pluginName: {"foo": "bar", "baz": "qux"},
		}
	})

	It("returns config for known plugin", func() {
		resp, err := svc.GetPluginConfig(context.Background(), &hostconfig.GetPluginConfigRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Config).To(HaveKeyWithValue("foo", "bar"))
		Expect(resp.Config).To(HaveKeyWithValue("baz", "qux"))
	})

	It("returns error for unknown plugin", func() {
		svc.pluginID = "unknown"
		resp, err := svc.GetPluginConfig(context.Background(), &hostconfig.GetPluginConfigRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Config).To(BeEmpty())
	})

	It("returns empty config if plugin config is empty", func() {
		conf.Server.PluginConfig[pluginName] = map[string]string{}
		resp, err := svc.GetPluginConfig(context.Background(), &hostconfig.GetPluginConfigRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Config).To(BeEmpty())
	})
})
