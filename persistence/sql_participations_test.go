package persistence

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sql_participations", func() {
	Describe("marshalParticipants/unmarshalParticipants", func() {
		It("preserves CreditedAs through marshal/unmarshal round-trip", func() {
			original := model.Participants{
				model.RoleArtist: model.ParticipantList{
					{Artist: model.Artist{ID: "canon-1", Name: "Planetary Assault Systems"}, CreditedAs: "PAS"},
				},
				model.RoleComposer: model.ParticipantList{
					{Artist: model.Artist{ID: "canon-2", Name: "John Lennon"}, CreditedAs: "J. Lennon"},
				},
			}
			serialized := marshalParticipants(original)
			restored, err := unmarshalParticipants(serialized)
			Expect(err).ToNot(HaveOccurred())
			Expect(restored[model.RoleArtist]).To(HaveLen(1))
			Expect(restored[model.RoleArtist][0].CreditedAs).To(Equal("PAS"))
			Expect(restored[model.RoleComposer][0].CreditedAs).To(Equal("J. Lennon"))
		})

		It("preserves SubRole through marshal/unmarshal round-trip", func() {
			original := model.Participants{
				model.RolePerformer: model.ParticipantList{
					{Artist: model.Artist{ID: "p-1", Name: "Performer"}, SubRole: "Guitar"},
				},
			}
			serialized := marshalParticipants(original)
			restored, err := unmarshalParticipants(serialized)
			Expect(err).ToNot(HaveOccurred())
			Expect(restored[model.RolePerformer]).To(HaveLen(1))
			Expect(restored[model.RolePerformer][0].SubRole).To(Equal("Guitar"))
		})

		It("does not include CreditedAs in serialized JSON when empty (omitempty)", func() {
			p := model.Participants{
				model.RoleArtist: model.ParticipantList{
					{Artist: model.Artist{ID: "canon-1", Name: "Foo"}},
				},
			}
			serialized := marshalParticipants(p)
			Expect(serialized).NotTo(ContainSubstring("creditedAs"))
		})
	})
})
