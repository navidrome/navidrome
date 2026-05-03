package tagwriter

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/go-flac/go-flac"
)

func writeFLACTags(filePath string, tags Tags) error {
	flacFile, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	var vorbisCommentIndex int = -1
	for i, block := range flacFile.Meta {
		if block.Type == flac.VorbisComment {
			vorbisCommentIndex = i
			break
		}
	}

	vorbisData := encodeVorbisComments(tags)

	if vorbisCommentIndex >= 0 {
		flacFile.Meta[vorbisCommentIndex].Data = vorbisData
	} else {
		flacFile.Meta = append(flacFile.Meta, &flac.MetaDataBlock{
			Type: flac.VorbisComment,
			Data: vorbisData,
		})
	}

	if err := flacFile.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

func encodeVorbisComments(tags Tags) flac.BlockData {
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

func countNonEmptyTags(tags Tags) int {
	count := 0
	for _, v := range tags {
		if v != "" {
			count++
		}
	}
	return count
}

func encodeUint32LE(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func init() {
	_ = os.Stdin
}