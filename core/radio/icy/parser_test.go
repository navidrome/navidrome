package icy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReadStreamTitles", func() {
	const metaInt = 5

	readTitles := func(ctx context.Context, stream []byte, interval int) ([]string, error) {
		var titles []string
		err := ReadStreamTitles(ctx, bytes.NewReader(stream), interval, func(title string) {
			titles = append(titles, title)
		})
		return titles, err
	}

	It("emits changed StreamTitle values", func() {
		stream := icyStream(metaInt,
			"StreamTitle='Artist One - Track One';",
			"StreamTitle='Artist One - Track One';",
			"StreamTitle='Artist Two - Track Two';",
		)

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(Equal([]string{
			"Artist One - Track One",
			"Artist Two - Track Two",
		}))
	})

	It("ignores empty metadata blocks", func() {
		stream := audioInterval(metaInt)
		stream = append(stream, 0)

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(BeEmpty())
	})

	It("ignores metadata blocks without StreamTitle", func() {
		stream := icyStream(metaInt, "StreamUrl='https://example.test';")

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(BeEmpty())
	})

	It("ignores empty StreamTitle values", func() {
		stream := icyStream(metaInt, "StreamTitle='';")

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(BeEmpty())
	})

	It("returns an error for invalid metadata intervals", func() {
		titles, err := readTitles(context.Background(), nil, 0)

		Expect(err).To(MatchError(ErrInvalidMetaInt))
		Expect(titles).To(BeEmpty())
	})

	It("stops when context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		titles, err := readTitles(ctx, audioInterval(metaInt), metaInt)

		Expect(err).To(MatchError(context.Canceled))
		Expect(titles).To(BeEmpty())
	})

	It("returns cleanly when input ends mid-audio interval", func() {
		titles, err := readTitles(context.Background(), []byte("abc"), metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(BeEmpty())
	})

	It("returns cleanly when input ends mid-metadata block", func() {
		stream := append(audioInterval(metaInt), 2, 'S', 't', 'r')

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(BeEmpty())
	})

	It("decodes Latin-1 metadata", func() {
		metadata := []byte("StreamTitle='Beyonc")
		metadata = append(metadata, 0xe9)
		metadata = append(metadata, " - D\xe9j\xe0 Vu';"...)
		stream := icyStreamBytes(metaInt, metadata)

		titles, err := readTitles(context.Background(), stream, metaInt)

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(Equal([]string{"Beyoncé - Déjà Vu"}))
	})

	It("returns reader errors other than EOF", func() {
		errReader := errAfterReader{
			data: []byte("abcde"),
			err:  errors.New("read failed"),
		}
		var titles []string

		err := ReadStreamTitles(context.Background(), &errReader, metaInt, func(title string) {
			titles = append(titles, title)
		})

		Expect(err).To(MatchError("read failed"))
		Expect(titles).To(BeEmpty())
	})
})

func icyStream(metaInt int, metadataBlocks ...string) []byte {
	blocks := make([][]byte, 0, len(metadataBlocks))
	for _, block := range metadataBlocks {
		blocks = append(blocks, []byte(block))
	}
	return icyStreamBytes(metaInt, blocks...)
}

func icyStreamBytes(metaInt int, metadataBlocks ...[]byte) []byte {
	var stream []byte
	for _, metadata := range metadataBlocks {
		stream = append(stream, audioInterval(metaInt)...)
		stream = append(stream, metadataBlock(metadata)...)
	}
	return stream
}

func audioInterval(metaInt int) []byte {
	return []byte(strings.Repeat("a", metaInt))
}

func metadataBlock(metadata []byte) []byte {
	if len(metadata) == 0 {
		return []byte{0}
	}
	blockCount := (len(metadata) + 15) / 16
	block := []byte{byte(blockCount)}
	block = append(block, metadata...)
	padding := blockCount*16 - len(metadata)
	block = append(block, bytes.Repeat([]byte{0}, padding)...)
	return block
}

type errAfterReader struct {
	data []byte
	err  error
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, r.err
}

var _ io.Reader = (*errAfterReader)(nil)
