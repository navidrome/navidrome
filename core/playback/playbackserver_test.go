package playback

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/playback/cast"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type castTrackFactoryCall struct {
	ctx          context.Context
	playbackDone chan bool
	target       cast.Target
	mf           model.MediaFile
}

type castTrackFactoryRecorder struct {
	calls []castTrackFactoryCall
	err   error
}

func (r *castTrackFactoryRecorder) factory(ctx context.Context, playbackDone chan bool, target cast.Target, mf model.MediaFile) (Track, error) {
	r.calls = append(r.calls, castTrackFactoryCall{ctx: ctx, playbackDone: playbackDone, target: target, mf: mf})
	if r.err != nil {
		return nil, r.err
	}
	return &fakeTrack{name: fmt.Sprintf("cast-%s", mf.ID)}, nil
}

var _ = Describe("playbackServer", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("preserves ordinary MPV targets byte-for-byte", func() {
		rec := &trackFactoryRecorder{}
		ps := &playbackServer{trackFactory: rec.factory}
		target := "  hw:1,0  "

		devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", target}}, "", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())
		Expect(devices).To(HaveLen(1))
		Expect(devices[0].DeviceName).To(Equal(target))

		devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))
		_, err = devices[0].Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].deviceName).To(Equal(target))
	})

	It("keeps non-exact cast prefixes on the MPV path", func() {
		rec := &trackFactoryRecorder{}
		ps := &playbackServer{trackFactory: rec.factory}

		devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "Cast:foo"}, {"kitchen", " CAST:foo"}}, "living", ps.trackFactory)
		Expect(err).ToNot(HaveOccurred())

		devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))
		_, err = devices[0].Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		devices[1].PlaybackQueue.Add(makeMediaFiles("track-2"))
		_, err = devices[1].Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(HaveLen(2))
		Expect(rec.calls[0].deviceName).To(Equal("Cast:foo"))
		Expect(rec.calls[1].deviceName).To(Equal(" CAST:foo"))
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

	Describe("Cast device wiring", func() {
		var oldCastTrackFactory func(context.Context, chan bool, cast.Target, model.MediaFile) (Track, error)

		BeforeEach(func() {
			oldCastTrackFactory = castTrackFactory
		})

		AfterEach(func() {
			castTrackFactory = oldCastTrackFactory
		})

		It("dispatches exact cast targets to the Cast factory", func() {
			rec := &castTrackFactoryRecorder{}
			castTrackFactory = rec.factory
			ps := &playbackServer{trackFactory: defaultTrackFactory}

			devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"Kitchen speaker", "cast:192.168.1.50:8009"}}, "", ps.trackFactory)
			Expect(err).ToNot(HaveOccurred())
			devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))

			_, err = devices[0].Start(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(rec.calls).To(HaveLen(1))
			Expect(rec.calls[0].target).To(Equal(cast.Target{Name: "Kitchen speaker", Host: "192.168.1.50", Port: 8009}))
			Expect(rec.calls[0].ctx).To(BeIdenticalTo(ctx))
			Expect(rec.calls[0].playbackDone).To(BeIdenticalTo(devices[0].PlaybackDone))
			Expect(rec.calls[0].mf.ID).To(Equal("track-1"))
		})

		It("captures separate Cast targets per configured device", func() {
			rec := &castTrackFactoryRecorder{}
			castTrackFactory = rec.factory
			ps := &playbackServer{trackFactory: defaultTrackFactory}

			devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "cast:192.168.1.50"}, {"kitchen", "cast:[2001:db8::10]:9000"}}, "living", ps.trackFactory)
			Expect(err).ToNot(HaveOccurred())
			devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))
			devices[1].PlaybackQueue.Add(makeMediaFiles("track-2"))

			_, err = devices[0].Start(ctx)
			Expect(err).ToNot(HaveOccurred())
			_, err = devices[1].Start(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(rec.calls).To(HaveLen(2))
			Expect(rec.calls[0].target).To(Equal(cast.Target{Name: "living", Host: "192.168.1.50", Port: 8009}))
			Expect(rec.calls[1].target).To(Equal(cast.Target{Name: "kitchen", Host: "2001:db8::10", Port: 9000}))
		})

		It("allows startup with a valid Cast device when BaseHost is unset and fails on first track creation", func() {
			oldBaseHost := conf.Server.BaseHost
			DeferCleanup(func() {
				conf.Server.BaseHost = oldBaseHost
			})
			conf.Server.BaseHost = ""

			ps := &playbackServer{trackFactory: defaultTrackFactory}
			devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "cast:192.168.1.50"}}, "", ps.trackFactory)
			Expect(err).ToNot(HaveOccurred())

			devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))
			_, err = devices[0].Start(ctx)
			Expect(err).To(MatchError("playback URL requires BaseHost"))
		})

		It("propagates Cast factory errors through track creation", func() {
			rec := &castTrackFactoryRecorder{err: errors.New("cast connect failed")}
			castTrackFactory = rec.factory
			ps := &playbackServer{trackFactory: defaultTrackFactory}

			devices, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "cast:192.168.1.50"}}, "", ps.trackFactory)
			Expect(err).ToNot(HaveOccurred())
			devices[0].PlaybackQueue.Add(makeMediaFiles("track-1"))

			_, err = devices[0].Start(ctx)
			Expect(err).To(MatchError("cast connect failed"))
		})
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

	It("does not fall through malformed cast targets to the MPV factory", func() {
		rec := &trackFactoryRecorder{}
		ps := &playbackServer{trackFactory: rec.factory}

		_, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", "cast:bad:target:shape"}}, "", ps.trackFactory)
		Expect(err).To(HaveOccurred())
		Expect(rec.calls).To(BeEmpty())
	})

	DescribeTable("rejects malformed cast targets at startup",
		func(target string) {
			ps := &playbackServer{trackFactory: defaultTrackFactory}
			_, err := ps.initDeviceStatus(ctx, []conf.AudioDeviceDefinition{{"living", target}}, "", ps.trackFactory)
			Expect(err).To(HaveOccurred())
		},
		Entry("empty target", "cast:"),
		Entry("empty host", "cast::8009"),
		Entry("non-numeric port", "cast:host:bad"),
		Entry("plus port", "cast:host:+8009"),
		Entry("minus port", "cast:host:-8009"),
		Entry("port with leading whitespace", "cast:host: 8009"),
		Entry("port with trailing whitespace", "cast:host:8009 "),
		Entry("port with separator", "cast:host:8_009"),
		Entry("port zero", "cast:host:0"),
		Entry("port too large", "cast:host:65536"),
		Entry("malformed IPv6 bracket", "cast:[2001:db8::1"),
		Entry("bracketed hostname", "cast:[chromecast.local]"),
		Entry("bracketed IPv4", "cast:[192.168.1.50]"),
		Entry("ambiguous unbracketed IPv6", "cast:2001:db8::10"),
		Entry("cast URL syntax", "cast://host"),
		Entry("missing IPv6 port digits", "cast:[2001:db8::10]:"),
		Entry("leading host whitespace", "cast: host"),
		Entry("trailing host whitespace", "cast:host "),
		Entry("internal host whitespace", "cast:chrome cast"),
		Entry("tab in host", "cast:\thost"),
		Entry("tab in hostname", "cast:host\tname"),
		Entry("reservation stays in cast namespace", "cast:bad:target:shape"),
	)

	DescribeTable("parses valid Cast targets at startup",
		func(target string, expected cast.Target) {
			parsed, err := parseCastTarget(expected.Name, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).To(Equal(expected))
		},
		Entry("IPv4 with default port", "cast:192.168.1.50", cast.Target{Name: "living", Host: "192.168.1.50", Port: 8009}),
		Entry("IPv4 with explicit port", "cast:192.168.1.50:8009", cast.Target{Name: "living", Host: "192.168.1.50", Port: 8009}),
		Entry("hostname with default port", "cast:chromecast.local", cast.Target{Name: "living", Host: "chromecast.local", Port: 8009}),
		Entry("bracketed IPv6 default port", "cast:[2001:db8::10]", cast.Target{Name: "living", Host: "2001:db8::10", Port: 8009}),
		Entry("bracketed IPv6 explicit port", "cast:[2001:db8::10]:9000", cast.Target{Name: "living", Host: "2001:db8::10", Port: 9000}),
		Entry("minimum port", "cast:host:1", cast.Target{Name: "living", Host: "host", Port: 1}),
		Entry("maximum port", "cast:host:65535", cast.Target{Name: "living", Host: "host", Port: 65535}),
	)

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
