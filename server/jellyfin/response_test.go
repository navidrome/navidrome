package jellyfin

import (
	"bytes"
	"encoding/json"
	"errors"
	"iter"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("streamQueryResult", func() {
	// The streamed output must be byte-for-byte identical to what json.Encoder.Encode produced
	// before, so no client sees a different response.
	assertIdenticalToEncoder := func(q dto.QueryResult) {
		var got bytes.Buffer
		Expect(streamQueryResult(&got, q)).To(Succeed())

		var want bytes.Buffer
		Expect(json.NewEncoder(&want).Encode(q)).To(Succeed())

		Expect(got.String()).To(Equal(want.String()))
	}

	It("encodes an empty item list", func() {
		assertIdenticalToEncoder(dto.QueryResult{Items: []dto.BaseItemDto{}})
	})

	It("encodes a single item", func() {
		assertIdenticalToEncoder(dto.QueryResult{
			Items:            []dto.BaseItemDto{{Id: "a", Name: "One"}},
			TotalRecordCount: 1,
		})
	})

	It("encodes multiple items, honoring HTML escaping and StartIndex", func() {
		assertIdenticalToEncoder(dto.QueryResult{
			Items: []dto.BaseItemDto{
				{Id: "a", Name: "One"},
				{Id: "b", Name: "Two & <Three>"},
			},
			TotalRecordCount: 500,
			StartIndex:       100,
		})
	})
})

var _ = Describe("streamItemsEnvelope", func() {
	seqOf := func(items ...dto.BaseItemDto) iter.Seq2[dto.BaseItemDto, error] {
		return func(yield func(dto.BaseItemDto, error) bool) {
			for _, it := range items {
				if !yield(it, nil) {
					return
				}
			}
		}
	}

	It("produces the same bytes as encoding an equivalent QueryResult", func() {
		items := []dto.BaseItemDto{{Id: "a", Name: "One"}, {Id: "b", Name: "Two & <Three>"}}
		var got bytes.Buffer
		Expect(streamItemsEnvelope(&got, seqOf(items...), 500, 100)).To(Succeed())

		var want bytes.Buffer
		Expect(json.NewEncoder(&want).Encode(dto.QueryResult{Items: items, TotalRecordCount: 500, StartIndex: 100})).To(Succeed())
		Expect(got.String()).To(Equal(want.String()))
	})

	It("emits an empty array (not null) for a sequence that yields nothing", func() {
		var got bytes.Buffer
		Expect(streamItemsEnvelope(&got, seqOf(), 0, 0)).To(Succeed())
		Expect(got.String()).To(Equal("{\"Items\":[],\"TotalRecordCount\":0,\"StartIndex\":0}\n"))
	})

	It("stops at a mid-stream error but still closes the envelope and returns the error", func() {
		boom := errors.New("scan failed")
		first := dto.BaseItemDto{Id: "a", Name: "One"}
		seq := func(yield func(dto.BaseItemDto, error) bool) {
			if !yield(first, nil) {
				return
			}
			yield(dto.BaseItemDto{}, boom)
		}
		var got bytes.Buffer
		err := streamItemsEnvelope(&got, seq, 7, 0)
		Expect(err).To(MatchError(boom))
		// Valid JSON with the item written before the failure, and the totals still present.
		firstJSON, _ := json.Marshal(first)
		Expect(got.String()).To(Equal("{\"Items\":[" + string(firstJSON) + "],\"TotalRecordCount\":7,\"StartIndex\":0}\n"))
	})
})
