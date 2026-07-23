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

	It("omits artwork state when the entity has none", func() {
		var out map[string]any
		data, err := json.Marshal(model.Album{ID: "al-2", Name: "Album"})
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, &out)).To(Succeed())

		Expect(out).ToNot(HaveKey("imageHash"))
		Expect(out).ToNot(HaveKey("blurHash"))
		Expect(out).ToNot(HaveKey("imageAbsent"))
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
