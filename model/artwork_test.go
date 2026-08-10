package model_test

import (
	"encoding/json"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func jsonOf(v any) map[string]any {
	GinkgoHelper()
	data, err := json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())
	var out map[string]any
	Expect(json.Unmarshal(data, &out)).To(Succeed())
	return out
}

var _ = Describe("ItemImage JSON", func() {
	It("exposes the artwork state a client needs to render a placeholder", func() {
		al := model.Album{ID: "al-1", Name: "Album"}
		al.ImageHash = "0123456789abcdef"
		al.ThumbHash = "1QcSHQRn"
		al.ImageWidth, al.ImageHeight = 1200, 800

		Expect(jsonOf(al)).To(SatisfyAll(
			HaveKeyWithValue("imageHash", "0123456789abcdef"),
			HaveKeyWithValue("thumbHash", "1QcSHQRn"),
			HaveKeyWithValue("imageWidth", BeNumerically("==", 1200)),
			HaveKeyWithValue("imageHeight", BeNumerically("==", 800)),
		))
	})

	It("keeps the blurhash off the native API, where nothing consumes it", func() {
		al := model.Album{ID: "al-1", Name: "Album"}
		al.BlurHash = "LEHV6nWB2yk8"
		Expect(jsonOf(al)).ToNot(HaveKey("blurHash"))
	})

	It("omits every artwork field when the entity has none", func() {
		out := jsonOf(model.Album{ID: "al-2", Name: "Album"})
		for _, key := range []string{
			"imageHash", "blurHash", "thumbHash", "imageAbsent", "imageWidth", "imageHeight",
		} {
			Expect(out).ToNot(HaveKey(key))
		}
	})

	It("exposes known-absent artwork so clients can skip the request", func() {
		ar := model.Artist{ID: "ar-1", Name: "Artist"}
		ar.ImageAbsent = true
		Expect(jsonOf(ar)).To(HaveKeyWithValue("imageAbsent", true))
	})

	Describe("AspectRatio", func() {
		It("returns width/height", func() {
			img := model.ItemImage{ImageHash: "abc", ImageWidth: 1200, ImageHeight: 800}
			Expect(*img.AspectRatio()).To(BeNumerically("~", 1.5, 0.0001))
		})

		It("returns nil when a dimension is missing, so callers never guess a ratio", func() {
			Expect(model.ItemImage{ImageHash: "abc"}.AspectRatio()).To(BeNil())
			Expect(model.ItemImage{ImageHash: "abc", ImageWidth: 1200}.AspectRatio()).To(BeNil())
			Expect(model.ItemImage{ImageHash: "abc", ImageHeight: 800}.AspectRatio()).To(BeNil())
		})

		It("returns nil for a known-absent image, whatever the dimensions say", func() {
			img := model.ItemImage{ImageAbsent: true, ImageWidth: 1200, ImageHeight: 800}
			Expect(img.AspectRatio()).To(BeNil())
		})
	})
})
