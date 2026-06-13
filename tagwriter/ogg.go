package tagwriter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

func writeOGGTags(filePath string, tags Tags) error {
	f, err := os.OpenFile(filePath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open OGG file: %w", err)
	}
	defer f.Close()

	header, err := readOGGPageHeader(f)
	if err != nil {
		return fmt.Errorf("invalid OGG file: %w", err)
	}

	if !bytes.Equal(header.Magic[:4], []byte("OggS")) {
		return errors.New("invalid OGG file: missing OGGS header")
	}

	pages, err := parseOGGPages(f)
	if err != nil {
		return fmt.Errorf("failed to parse OGG pages: %w", err)
	}

	vorbisCommentPage, commentSegment, err := findVorbisCommentPage(pages, f)
	if err != nil {
		return fmt.Errorf("failed to find Vorbis comment: %w", err)
	}

	vorbisData := encodeVorbisCommentsOgg(tags)

	if len(vorbisData) == 0 {
		return nil
	}

	if vorbisCommentPage != nil {
		if err := updateVorbisComment(f, vorbisCommentPage, commentSegment, vorbisData); err != nil {
			return fmt.Errorf("failed to update Vorbis comment: %w", err)
		}
	} else {
		if err := insertVorbisComment(f, header, vorbisData); err != nil {
			return fmt.Errorf("failed to insert Vorbis comment: %w", err)
		}
	}

	recalculateOGGChecksums(f)

	updateFileTimes(filePath)

	return nil
}

type oggPageHeader struct {
	Magic       [4]byte
	Version     byte
	HeaderType  byte
	GranulePos  uint64
	Serial      uint32
	PageSeq     uint32
	Checksum    uint32
	PageSegments byte
}

type oggPage struct {
	Header  oggPageHeader
	Offset  int64
	SegmentSizes []byte
	SegmentsStart int64
	DataStart int64
}

func readOGGPageHeader(f *os.File) (oggPageHeader, error) {
	header := make([]byte, 27)
	_, err := f.Read(header)
	if err != nil {
		return oggPageHeader{}, err
	}

	var h oggPageHeader
	copy(h.Magic[:], header[0:4])
	h.Version = header[4]
	h.HeaderType = header[5]
	h.GranulePos = binary.LittleEndian.Uint64(header[6:14])
	h.Serial = binary.LittleEndian.Uint32(header[14:18])
	h.PageSeq = binary.LittleEndian.Uint32(header[18:22])
	h.Checksum = binary.LittleEndian.Uint32(header[22:26])
	h.PageSegments = header[26]

	return h, nil
}

func parseOGGPages(f *os.File) ([]oggPage, error) {
	var pages []oggPage
	offset := int64(0)

	for {
		header, err := readOGGPageHeaderAt(f, offset)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		segmentSizes := make([]byte, header.PageSegments)
		if _, err := f.ReadAt(segmentSizes, offset+27); err != nil {
			return nil, err
		}

		segmentsStart := offset + 27 + int64(header.PageSegments)
		dataStart := segmentsStart

		var totalDataSize int64
		for _, segSize := range segmentSizes {
			totalDataSize += int64(segSize)
		}

		page := oggPage{
			Header:         header,
			Offset:         offset,
			SegmentSizes:   segmentSizes,
			SegmentsStart:  segmentsStart,
			DataStart:      dataStart,
		}
		pages = append(pages, page)

		pageSize := segmentsStart + totalDataSize - offset
		offset += pageSize

		if pageSize == 0 {
			break
		}
	}

	return pages, nil
}

func readOGGPageHeaderAt(f *os.File, offset int64) (oggPageHeader, error) {
	header := make([]byte, 27)
	_, err := f.ReadAt(header, offset)
	if err != nil {
		return oggPageHeader{}, err
	}

	var h oggPageHeader
	copy(h.Magic[:], header[0:4])
	h.Version = header[4]
	h.HeaderType = header[5]
	h.GranulePos = binary.LittleEndian.Uint64(header[6:14])
	h.Serial = binary.LittleEndian.Uint32(header[14:18])
	h.PageSeq = binary.LittleEndian.Uint32(header[18:22])
	h.Checksum = binary.LittleEndian.Uint32(header[22:26])
	h.PageSegments = header[26]

	return h, nil
}

func findVorbisCommentPage(pages []oggPage, f *os.File) (*oggPage, int, error) {
	for i, page := range pages {
		if page.Header.HeaderType&0x02 == 0 {
			continue
		}

		if len(page.SegmentSizes) == 0 {
			continue
		}

		data := make([]byte, page.SegmentSizes[0])
		if _, err := f.ReadAt(data, page.DataStart); err != nil {
			continue
		}

		if len(data) >= 7 && bytes.Equal(data[0:7], []byte("vorbis")) {
			return &pages[i], 0, nil
		}

		var cumulative int
		for segIdx, segSize := range page.SegmentSizes {
			cumulative += int(segSize)
			if cumulative >= 7 {
				headerData := make([]byte, segSize)
				readOffset := page.DataStart + int64(cumulative - int(segSize))
				f.ReadAt(headerData, readOffset)
				if bytes.Equal(headerData[:7], []byte("vorbis")) {
					return &pages[i], segIdx, nil
				}
				break
			}
		}
	}

	return nil, 0, errors.New("Vorbis comment header not found - creating new header")

}

func encodeVorbisCommentsOgg(tags Tags) []byte {
	buf := make([]byte, 0)

	vendor := "Navidrome"
	vendorBytes := []byte(vendor)
	buf = append(buf, encodeUint32LE(uint32(len(vendorBytes)))...)
	buf = append(buf, vendorBytes...)

	numComments := countNonEmptyTags(tags)
	buf = append(buf, encodeUint32LE(uint32(numComments))...)

	commentPairs := map[string]string{
		"TITLE":       TagTitle,
		"ARTIST":      TagArtist,
		"ALBUM":       TagAlbum,
		"ALBUMARTIST": TagAlbumArtist,
		"DATE":        TagYear,
		"YEAR":        TagYear,
		"GENRE":       TagGenre,
		"TRACKNUMBER": TagTrackNumber,
		"TRACKTOTAL":  TagTrackTotal,
		"DISCNUMBER":  TagDiscNumber,
		"DISCTOTAL":   TagDiscTotal,
		"COMMENT":     TagComment,
	}

	for vorbisKey, tagKey := range commentPairs {
		if value, ok := tags[tagKey]; ok && value != "" {
			comment := fmt.Sprintf("%s=%s", vorbisKey, value)
			commentBytes := []byte(comment)
			buf = append(buf, encodeUint32LE(uint32(len(commentBytes)))...)
			buf = append(buf, commentBytes...)
		}
	}

	return buf
}

func updateVorbisComment(f *os.File, page *oggPage, segmentIdx int, vorbisData []byte) error {
	pageDataSize := int64(0)
	for _, segSize := range page.SegmentSizes {
		pageDataSize += int64(segSize)
	}

	oldDataSize := int64(0)
	for i := segmentIdx; i < len(page.SegmentSizes); i++ {
		oldDataSize += int64(page.SegmentSizes[i])
	}

	delta := int64(len(vorbisData)) - oldDataSize

	if delta == 0 {
		dataOffset := page.DataStart + pageDataSize - oldDataSize
		_, err := f.WriteAt(vorbisData, dataOffset)
		return err
	}

	if delta > 0 {
		fileSize, err := f.Seek(0, os.SEEK_END)
		if err != nil {
			return err
		}

		pageEnd := page.Offset + 27 + int64(page.Header.PageSegments) + pageDataSize

		moveBuf := make([]byte, 4096)
		for pos := fileSize - 4096; pos >= pageEnd; pos -= 4096 {
			_, err := f.ReadAt(moveBuf, pos)
			if err != nil {
				return err
			}
			_, err = f.WriteAt(moveBuf, pos+delta)
			if err != nil {
				return err
			}
		}

		if fileSize-pageEnd < 4096 {
			remaining := make([]byte, fileSize-pageEnd)
			f.ReadAt(remaining, pageEnd)
			f.WriteAt(remaining, pageEnd+delta)
		}
	}

	dataOffset := page.DataStart
	_, err := f.WriteAt(vorbisData, dataOffset)
	return err
}

func insertVorbisComment(f *os.File, firstPage oggPageHeader, vorbisData []byte) error {
	commentHeader := createVorbisCommentHeader(vorbisData)

	commentData := append(commentHeader, vorbisData...)

	newFirstPage := firstPage
	newFirstPage.HeaderType |= 0x01

	newPageSize := 27 + 1 + int64(len(commentData))

	pageData := make([]byte, 0, newPageSize)
	pageData = append(pageData, []byte("OggS")...)
	pageData = append(pageData, newFirstPage.Version)
	pageData = append(pageData, newFirstPage.HeaderType)
	pageData = append(pageData, make([]byte, 8)...)
	serialBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(serialBytes, newFirstPage.Serial)
	pageData = append(pageData, serialBytes...)
	seqBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(seqBytes, newFirstPage.PageSeq)
	pageData = append(pageData, seqBytes...)
	pageData = append(pageData, make([]byte, 4)...)
	pageData = append(pageData, 1)
	pageData = append(pageData, byte(len(commentData)))
	pageData = append(pageData, commentData...)

	_, err := f.WriteAt(pageData, 0)
	return err
}

func createVorbisCommentHeader(data []byte) []byte {
	header := make([]byte, 7)
	copy(header, []byte("vorbis"))
	return header
}

func recalculateOGGChecksums(f *os.File) error {
	offset := int64(0)

	for {
		header := make([]byte, 27)
		n, err := f.ReadAt(header, offset)
		if err != nil || n < 27 {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		if !bytes.Equal(header[0:4], []byte("OggS")) {
			break
		}

		pageSegments := header[26]
		segmentSizes := make([]byte, pageSegments)
		f.ReadAt(segmentSizes, offset+27)

		var pageSize int64 = 27 + int64(pageSegments)
		for _, segSize := range segmentSizes {
			pageSize += int64(segSize)
		}

		pageData := make([]byte, pageSize)
		f.ReadAt(pageData, offset)

		checksum := computeCRC(pageData)
		checksumBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(checksumBytes, checksum)

		f.WriteAt(checksumBytes, offset+22)

		offset += pageSize
		if offset <= 0 {
			break
		}
	}

	return nil
}

func computeCRC(data []byte) uint32 {
	crcTable := make([]uint32, 256)
	for i := range crcTable {
		c := uint32(i)
		for j := 0; j < 8; j++ {
			if c&1 != 0 {
				c = 0xedb88320 ^ (c >> 1)
			} else {
				c = c >> 1
			}
		}
		crcTable[i] = c
	}

	var crc uint32 = 0xffffffff
	for _, b := range data {
		crc = crcTable[byte(crc)^b] ^ (crc >> 8)
	}
	return crc ^ 0xffffffff
}

func init() {
	_ = os.Stdin
}