package model_test

import (
	"encoding/json"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ItemImage JSON", func() {
	It("exposes artwork state on an album", func() {
		al := model.Album{ID: "al-1", Name: "Album"}
		al.ImageHash = "0123456789abcdef"
		al.BlurHash = "LEHV6nWB2yk8"

		var out map[string]any
		data, err := json.Marshal(al)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, &out)).To(Succeed())

		Expect(out).To(HaveKeyWithValue("imageHash", "0123456789abcdef"))
		Expect(out).To(HaveKeyWithValue("blurHash", "LEHV6nWB2yk8"))
	})

	// Without the dimensions, clients cannot know the placeholder's shape and default to a square.
	It("exposes the image dimensions alongside the blurhash", func() {
		al := model.Album{ID: "al-3", Name: "Album"}
		al.BlurHash = "LEHV6nWB2yk8"
		al.ImageWidth, al.ImageHeight = 1200, 800

		var out map[string]any
		data, err := json.Marshal(al)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, &out)).To(Succeed())

		Expect(out).To(HaveKeyWithValue("imageWidth", BeNumerically("==", 1200)))
		Expect(out).To(HaveKeyWithValue("imageHeight", BeNumerically("==", 800)))
	})

	It("omits artwork state when the entity has none", func() {
		var out map[string]any
		data, err := json.Marshal(model.Album{ID: "al-2", Name: "Album"})
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, &out)).To(Succeed())

		Expect(out).ToNot(HaveKey("imageHash"))
		Expect(out).ToNot(HaveKey("blurHash"))
		Expect(out).ToNot(HaveKey("imageAbsent"))
		Expect(out).ToNot(HaveKey("imageWidth"))
		Expect(out).ToNot(HaveKey("imageHeight"))
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

	It("exposes known-absent artwork so clients can skip the request", func() {
		ar := model.Artist{ID: "ar-1", Name: "Artist"}
		ar.ImageAbsent = true

		var out map[string]any
		data, err := json.Marshal(ar)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, &out)).To(Succeed())

		Expect(out).To(HaveKeyWithValue("imageAbsent", true))
	})
})
