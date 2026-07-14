package criteria

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("fields", func() {
	Describe("LookupField", func() {
		It("finds built-in fields case-insensitively", func() {
			field, ok := LookupField("Title")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("title"))
		})

		It("resolves aliases to their canonical field name", func() {
			field, ok := LookupField("albumtype")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("releasetype"))
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
		})

		It("finds registered tag names", func() {
			AddTagNames([]string{"task3_mood"})

			field, ok := LookupField("task3_mood")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("task3_mood"))
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
		})

		It("marks registered numeric tags", func() {
			AddTagNames([]string{"task3_score"})
			AddNumericTags([]string{"task3_score"})

			field, ok := LookupField("task3_score")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
			gomega.Expect(field.Numeric).To(gomega.BeTrue())
		})

		It("finds registered roles", func() {
			AddRoles([]string{"task3_producer"})

			field, ok := LookupField("task3_producer")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("task3_producer"))
			gomega.Expect(field.IsRole).To(gomega.BeTrue())
		})

		It("marks ReplayGain column fields as nullable", func() {
			field, ok := LookupField("rgAlbumGain")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("rgalbumgain"))
			gomega.Expect(field.Nullable).To(gomega.BeTrue())
			gomega.Expect(field.IsTag).To(gomega.BeFalse())
		})

		It("resolves replaygain_* tag names as aliases to nullable column fields", func() {
			// AddTagNames skips names already in the field map, so the startup tag registration
			// (from mappings.yaml) must not convert the pre-registered alias into a tag field.
			AddTagNames([]string{"replaygain_album_gain"})

			field, ok := LookupField("replaygain_album_gain")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("rgalbumgain"))
			gomega.Expect(field.Nullable).To(gomega.BeTrue())
			gomega.Expect(field.IsTag).To(gomega.BeFalse())
		})

		It("marks mbz_* and lyrics string fields as nullable (empty means missing)", func() {
			for _, name := range []string{"mbz_album_id", "mbz_album_artist_id", "mbz_artist_id",
				"mbz_recording_id", "mbz_release_track_id", "mbz_release_group_id", "lyrics",
				"album", "comment", "catalognumber", "discsubtitle", "albumcomment",
				"sorttitle", "sortalbum", "sortartist", "sortalbumartist", "explicitstatus"} {
				field, ok := LookupField(name)
				gomega.Expect(ok).To(gomega.BeTrue(), name)
				gomega.Expect(field.Nullable).To(gomega.BeTrue(), name)
				gomega.Expect(field.Numeric).To(gomega.BeFalse(), name)
			}
		})

	})
})
