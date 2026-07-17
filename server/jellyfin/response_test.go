package jellyfin

import (
	"bytes"
	"encoding/json"
	"errors"
	"iter"
	"strings"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// deadWriter stands in for a client that went away mid-response.
type deadWriter struct{}

func (deadWriter) Write([]byte) (int, error) { return 0, errors.New("connection reset by peer") }

var _ = Describe("streaming a materialized QueryResult", func() {
	// The materialized path (api.ok -> writeItems -> sliceItems) must stay byte-for-byte identical to
	// what json.Encoder.Encode produced before, so no client sees a different response.
	assertIdenticalToEncoder := func(q dto.QueryResult) {
		var got bytes.Buffer
		Expect(streamItemsEnvelope(&got, sliceItems(q.Items), q.TotalRecordCount, q.StartIndex)).To(Succeed())

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

	// A client that goes away must not keep the source (a DB cursor, holding its pooled connection
	// and a stream slot) running to the end of the library.
	It("stops pulling from the source once writing fails", func() {
		const total = 20000
		pulled := 0
		seq := func(yield func(dto.BaseItemDto, error) bool) {
			for range total {
				pulled++
				if !yield(dto.BaseItemDto{Id: "a", Name: strings.Repeat("x", 200)}, nil) {
					return
				}
			}
		}
		err := streamItemsEnvelope(deadWriter{}, seq, total, 0)
		Expect(err).To(HaveOccurred())
		Expect(pulled).To(BeNumerically("<", total), "should abandon the scan, not drain it")
	})

	It("aborts on a mid-stream error, leaving the envelope open (malformed) so the client fails loudly", func() {
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
		firstJSON, _ := json.Marshal(first)
		Expect(got.String()).To(Equal("{\"Items\":[" + string(firstJSON)))
	})
})
