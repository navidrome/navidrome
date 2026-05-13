package tagwriter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

func writeWAVTags(filePath string, tags Tags) error {
	f, err := os.OpenFile(filePath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer f.Close()

	riffHeader := make([]byte, 12)
	if _, err := f.Read(riffHeader); err != nil {
		return fmt.Errorf("failed to read RIFF header: %w", err)
	}

	if !bytes.Equal(riffHeader[:4], []byte("RIFF")) {
		return errors.New("invalid WAV file: missing RIFF header")
	}
	if !bytes.Equal(riffHeader[8:12], []byte("WAVE")[:4]) {
		return fmt.Errorf("invalid WAV file: expected WAVE format, found %q", string(riffHeader[8:12]))
	}

	chunks, err := parseRIFFChunks(f)
	if err != nil {
		return fmt.Errorf("failed to parse RIFF chunks: %w", err)
	}

	id3Chunk := findOrCreateID3Chunk(chunks)

	id3Data, err := encodeID3v2Tags(tags)
	if err != nil {
		return fmt.Errorf("failed to encode ID3v2 tags: %w", err)
	}

	if len(id3Data) == 0 {
		return nil
	}

	if id3Chunk != nil {
		chunkEnd := id3Chunk.Offset + 8 + int64(id3Chunk.Size)
		if id3Chunk.Size%2 != 0 {
			chunkEnd++
		}
		if err := f.Truncate(chunkEnd); err != nil {
			return fmt.Errorf("failed to truncate file: %w", err)
		}
	}

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	if err := writeRIFFChunk(f, []byte("id3 "), id3Data); err != nil {
		return fmt.Errorf("failed to write id3 chunk: %w", err)
	}

	if err := updateRIFFSize(f); err != nil {
		return fmt.Errorf("failed to update RIFF size: %w", err)
	}

	return nil
}

type riffChunk struct {
	ID      [4]byte
	Size    uint32
	Offset  int64
}

func parseRIFFChunks(f *os.File) ([]riffChunk, error) {
	var chunks []riffChunk
	offset := int64(12)

	for {
		chunkHeader := make([]byte, 8)
		n, err := f.ReadAt(chunkHeader, offset)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if n < 8 {
			break
		}

		var chunk riffChunk
		copy(chunk.ID[:], chunkHeader[:4])
		chunk.Size = binary.LittleEndian.Uint32(chunkHeader[4:8])
		chunk.Offset = offset

		chunks = append(chunks, chunk)

		padding := chunk.Size
		if padding%2 != 0 {
			padding++
		}
		offset += 8 + int64(padding)
	}

	return chunks, nil
}

func findOrCreateID3Chunk(chunks []riffChunk) *riffChunk {
	for i := range chunks {
		if bytes.Equal(chunks[i].ID[:], []byte("id3 ")) {
			return &chunks[i]
		}
	}
	return nil
}

func encodeID3v2Tags(tags Tags) ([]byte, error) {
	frames := bytes.NewBuffer(nil)

	if title, ok := tags[TagTitle]; ok && title != "" {
		frames.Write(createTextFrame("TIT2", title))
	}

	if artist, ok := tags[TagArtist]; ok && artist != "" {
		frames.Write(createTextFrame("TPE1", artist))
	}

	if album, ok := tags[TagAlbum]; ok && album != "" {
		frames.Write(createTextFrame("TALB", album))
	}

	if albumArtist, ok := tags[TagAlbumArtist]; ok && albumArtist != "" {
		frames.Write(createTextFrame("TPE2", albumArtist))
	}

	if year, ok := tags[TagYear]; ok && year != "" {
		frames.Write(createTextFrame("TYER", year))
	}

	if genre, ok := tags[TagGenre]; ok && genre != "" {
		frames.Write(createTextFrame("TCON", genre))
	}

	if trackNum, ok := tags[TagTrackNumber]; ok && trackNum != "" {
		trackTotal, _ := tags[TagTrackTotal]
		trackFrame := fmt.Sprintf("%s/%s", trackNum, trackTotal)
		frames.Write(createTextFrame("TRCK", trackFrame))
	}

	if discNum, ok := tags[TagDiscNumber]; ok && discNum != "" {
		discTotal, _ := tags[TagDiscTotal]
		discFrame := fmt.Sprintf("%s/%s", discNum, discTotal)
		frames.Write(createTextFrame("TPOS", discFrame))
	}

	if comment, ok := tags[TagComment]; ok && comment != "" {
		frames.Write(createCommentFrame(comment))
	}

	if frames.Len() == 0 {
		return nil, nil
	}

	tagSize := syncUint32(uint32(frames.Len()))

	header := make([]byte, 10)
	copy(header[0:3], []byte("ID3"))
	header[3] = 0x03
	header[4] = 0x00
	header[5] = 0x00
	copy(header[6:10], tagSize)

	result := bytes.NewBuffer(header)
	result.Write(frames.Bytes())

	return result.Bytes(), nil
}

func createTextFrame(frameID string, text string) []byte {
	textData := append([]byte{0x03}, []byte(text)...)

	frame := make([]byte, 10)
	copy(frame[0:4], []byte(frameID))
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(textData)))
	frame[8] = 0x00
	frame[9] = 0x00

	return append(frame, textData...)
}

func createCommentFrame(text string) []byte {
	frameData := new(bytes.Buffer)

	frameData.WriteByte(0x03)
	frameData.WriteString("eng")
	frameData.WriteByte(0x00)
	frameData.WriteString("")
	frameData.WriteByte(0x00)
	frameData.WriteString(text)

	dataLen := frameData.Len()

	frame := make([]byte, 10)
	copy(frame[0:4], []byte("COMM"))
	binary.BigEndian.PutUint32(frame[4:8], uint32(dataLen))
	frame[8] = 0x00
	frame[9] = 0x00

	return append(frame, frameData.Bytes()...)
}

func syncUint32(n uint32) []byte {
	result := make([]byte, 4)
	result[0] = byte((n >> 21) & 0x7F)
	result[1] = byte((n >> 14) & 0x7F)
	result[2] = byte((n >> 7) & 0x7F)
	result[3] = byte(n & 0x7F)
	return result
}

func writeRIFFChunk(f *os.File, id []byte, data []byte) error {
	chunk := make([]byte, 8)
	copy(chunk[:4], id)
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(data)))

	if _, err := f.Write(chunk); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}

	if len(data)%2 != 0 {
		if _, err := f.Write([]byte{0}); err != nil {
			return err
		}
	}

	return nil
}

func updateRIFFSize(f *os.File) error {
	fileSize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	riffSize := uint32(fileSize - 8)
	if riffSize%2 != 0 {
		riffSize++
	}

	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, riffSize)

	if _, err := f.WriteAt(sizeBytes, 4); err != nil {
		return err
	}

	return nil
}

func init() {
	_ = os.Stdin
}