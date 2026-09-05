package playback

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("playbackServer", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("injects the configured factory into created devices", func() {
		rec := &trackFactoryRecorder{}
		ps := &playbackServer{trackFactory: rec.factory}

		devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "hw:0"}, {"kitchen", "hw:1"}}, "living", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())
		Expect(devices).To(HaveLen(2))

		devices[1].PlaybackQueue.Add(makeMediaFiles("track-1"))
		_, err = devices[1].Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].deviceName).To(Equal("hw:1"))
		Expect(rec.calls[0].playbackDone).To(BeIdenticalTo(devices[1].PlaybackDone))
		Expect(rec.calls[0].mf.ID).To(Equal("track-1"))
	})

	It("creates the synthetic default device when no devices are configured", func() {
		ps := &playbackServer{trackFactory: defaultTrackFactory}

		devices, err := ps.initDeviceStatus(ctx, nil, "", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())
		Expect(devices).To(HaveLen(1))
		Expect(devices[0].Name).To(Equal("auto"))
		Expect(devices[0].DeviceName).To(Equal("auto"))
		Expect(devices[0].Default).To(BeTrue())
		Expect(devices[0].trackFactory).ToNot(BeNil())
	})

	It("creates a single configured device as default when no explicit default is set", func() {
		ps := &playbackServer{trackFactory: defaultTrackFactory}

		devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"office", "hw:2"}}, "", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())
		Expect(devices).To(HaveLen(1))
		Expect(devices[0].Name).To(Equal("office"))
		Expect(devices[0].DeviceName).To(Equal("hw:2"))
		Expect(devices[0].Default).To(BeTrue())
	})

	It("creates configured devices and preserves default target selection behavior", func() {
		ps := &playbackServer{trackFactory: defaultTrackFactory}
		devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "hw:0"}, {"kitchen", "hw:1"}}, "kitchen", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())
		ps.playbackDevices = devices

		device, err := ps.GetDeviceForUser("alice")
		Expect(err).ToNot(HaveOccurred())
		Expect(device.Name).To(Equal("kitchen"))
		Expect(device.DeviceName).To(Equal("hw:1"))
		Expect(device.Default).To(BeTrue())
		Expect(device.User).To(Equal("alice"))
	})

	It("wires the default track factory during Run when none is preset", func() {
		oldDevices := conf.Server.Jukebox.Devices
		oldDefault := conf.Server.Jukebox.Default
		DeferCleanup(func() {
			conf.Server.Jukebox.Devices = oldDevices
			conf.Server.Jukebox.Default = oldDefault
		})
		conf.Server.Jukebox.Devices = nil
		conf.Server.Jukebox.Default = ""

		ps := &playbackServer{datastore: &tests.MockDataStore{}}
		runCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := ps.Run(runCtx)
		Expect(err).ToNot(HaveOccurred())
		Expect(ps.trackFactory).ToNot(BeNil())
		Expect(ps.playbackDevices).To(HaveLen(1))
		Expect(ps.playbackDevices[0].trackFactory).ToNot(BeNil())
	})
})
