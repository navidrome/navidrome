package scanner

import (
	"context"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseFolders", func() {
	Describe("Artist re-PID migration", func() {
		var (
			artRepo *tests.MockArtistRepo
			p       *phaseFolders
		)

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			artRepo = tests.CreateMockArtistRepo()
			if artRepo.ReassignAnnotationCalls == nil {
				artRepo.ReassignAnnotationCalls = map[string]string{}
			}
			p = &phaseFolders{ctx: context.Background(), state: &scanState{}}
		})

		It("calls ReassignAnnotation when persisting an artist whose ID changed", func() {
			oldID := "old-id"
			newID := "new-id"
			idMap := map[string]string{newID: oldID}
			artist := model.Artist{ID: newID, Name: "Foo"}

			Expect(p.persistArtist(artRepo, &artist, idMap)).To(Succeed())
			Expect(artRepo.ReassignAnnotationCalls).To(HaveKeyWithValue(oldID, newID))
			// Mapping should be removed after successful reassignment
			Expect(idMap).ToNot(HaveKey(newID))
		})

		It("does not call ReassignAnnotation when no mapping exists", func() {
			idMap := map[string]string{}
			artist := model.Artist{ID: "some-id", Name: "Foo"}

			Expect(p.persistArtist(artRepo, &artist, idMap)).To(Succeed())
			Expect(artRepo.ReassignAnnotationCalls).To(BeEmpty())
		})

		It("calls CopyAttributes(created_at) when persisting an artist whose ID changed", func() {
			oldID := "old-id"
			newID := "new-id"
			idMap := map[string]string{newID: oldID}
			artist := model.Artist{ID: newID, Name: "Foo"}

			Expect(p.persistArtist(artRepo, &artist, idMap)).To(Succeed())
			Expect(artRepo.CopyAttributesCalls).To(HaveKeyWithValue(oldID, newID))
		})

		It("does not call CopyAttributes when no mapping exists", func() {
			idMap := map[string]string{}
			artist := model.Artist{ID: "some-id", Name: "Foo"}

			Expect(p.persistArtist(artRepo, &artist, idMap)).To(Succeed())
			Expect(artRepo.CopyAttributesCalls).To(BeEmpty())
		})
	})
})
