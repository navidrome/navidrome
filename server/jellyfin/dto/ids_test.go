package dto

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/model/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("id codec", func() {
	Describe("canonical ids", func() {
		It("round-trips a random id", func() {
			ndID := id.NewRandom()
			Expect(DecodeID(EncodeID(ndID))).To(Equal(ndID))
		})

		It("round-trips a hash id", func() {
			ndID := id.NewHash("artist", "Weird Al")
			Expect(DecodeID(EncodeID(ndID))).To(Equal(ndID))
		})

		It("emits exactly 32 lowercase hex chars", func() {
			for range 20 {
				Expect(EncodeID(id.NewRandom())).To(MatchRegexp("^[0-9a-f]{32}$"))
			}
		})

		It("accepts a dashed GUID, matching Jellyfin's Guid.Parse", func() {
			ndID := id.NewRandom()
			guid := EncodeID(ndID)
			dashed := guid[0:8] + "-" + guid[8:12] + "-" + guid[12:16] + "-" + guid[16:20] + "-" + guid[20:32]
			Expect(DecodeID(dashed)).To(Equal(ndID))
		})

		It("accepts an uppercase GUID", func() {
			ndID := id.NewRandom()
			Expect(DecodeID(strings.ToUpper(EncodeID(ndID)))).To(Equal(ndID))
		})
	})

	Describe("reserved space", func() {
		DescribeTable("round-trips library ids",
			func(libID int, guid string) {
				Expect(EncodeLibraryID(libID)).To(Equal(guid))
				Expect(DecodeID(guid)).To(Equal(strconv.Itoa(libID)))
			},
			Entry("first library", 1, "00000000000000000000000001000001"),
			Entry("double digit", 42, "0000000000000000000000000100002a"),
			Entry("max payload", 0xffffff, "00000000000000000000000001ffffff"),
		)

		It("round-trips the playlists folder", func() {
			Expect(PlaylistsFolderGUID).To(Equal("00000000000000000000000002000000"))
			Expect(DecodeID(PlaylistsFolderGUID)).To(Equal(PlaylistsFolderID))
		})

		DescribeTable("round-trips playlist entry positions",
			func(entryID, guid string) {
				Expect(EncodePlaylistEntryID(entryID)).To(Equal(guid))
				decoded, ok := DecodePlaylistEntryID(guid)
				Expect(ok).To(BeTrue())
				Expect(decoded).To(Equal(entryID))
			},
			Entry("first entry", "1", "00000000000000000000000003000001"),
			Entry("later entry", "300", "0000000000000000000000000300012c"),
		)

		It("rejects a non-integer playlist entry id", func() {
			Expect(EncodePlaylistEntryID("s1")).To(Equal(""))
			Expect(EncodePlaylistEntryID("")).To(Equal(""))
			Expect(EncodePlaylistEntryID("-1")).To(Equal(""))
		})

		It("rejects a payload wider than the reserved 24 bits", func() {
			Expect(EncodeLibraryID(1 << 24)).To(Equal(""))
			Expect(EncodeLibraryID(-1)).To(Equal(""))
			Expect(EncodePlaylistEntryID("16777216")).To(Equal(""))
		})

		It("keeps library and playlist-entry GUIDs distinct for the same number", func() {
			Expect(EncodeLibraryID(3)).ToNot(Equal(EncodePlaylistEntryID("3")))
		})

		It("does not let one reserved kind decode as another", func() {
			Expect(DecodeID(EncodePlaylistEntryID("3"))).To(Equal(""))
			_, ok := DecodePlaylistEntryID(EncodeLibraryID(3))
			Expect(ok).To(BeFalse())
			_, ok = DecodePlaylistEntryID(PlaylistsFolderGUID)
			Expect(ok).To(BeFalse())
		})

		It("rejects an unknown kind tag", func() {
			Expect(DecodeID("000000000000000000000000ff000000")).To(Equal(""))
		})

		It("never emits the all-zero GUID, which Jellyfin serializes as null", func() {
			Expect(EncodeLibraryID(0)).ToNot(Equal("00000000000000000000000000000000"))
			Expect(PlaylistsFolderGUID).ToNot(Equal("00000000000000000000000000000000"))
		})
	})

	Describe("malformed input", func() {
		DescribeTable("decodes to the empty string",
			func(input string) {
				Expect(DecodeID(input)).To(Equal(""))
			},
			Entry("empty", ""),
			Entry("too short", "abc123"),
			Entry("too long", "000000000000000000000000010000011"),
			Entry("non-hex chars", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"),
			Entry("a raw base62 id", id.NewRandom()),
			Entry("the old 44-char hex format", hex.EncodeToString([]byte("5QFKvMsJrd57QE2Le2dKKo"))),
		)

		It("encodes a non-canonical id to the empty string", func() {
			Expect(EncodeID("")).To(Equal(""))
			Expect(EncodeID("playlists")).To(Equal(""))
			Expect(EncodeID("42")).To(Equal(""))
		})
	})
})
