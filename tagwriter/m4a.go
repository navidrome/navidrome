package tagwriter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

func writeM4ATags(filePath string, tags Tags) error {
	f, err := os.OpenFile(filePath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open M4A file: %w", err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	originalSize := fileInfo.Size()

	atoms, err := parseMP4Atoms(f)
	if err != nil {
		return fmt.Errorf("failed to parse MP4 atoms: %w", err)
	}

	ilstAtom := findILSTAtom(atoms)

	metadataData, err := encodeILSTMetadata(tags)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	if len(metadataData) == 0 {
		return nil
	}

	newFileSize := originalSize
	if ilstAtom != nil {
		oldILSTSize := calculateAtomSize(int(ilstAtom.DataSize))
		newILSTSize := calculateAtomSize(len(metadataData))
		delta := int64(newILSTSize) - int64(oldILSTSize)
		newFileSize = originalSize + delta
	} else {
		newFileSize = originalSize + int64(calculateAtomSize(len(metadataData))+8)
	}

	if newFileSize > originalSize {
		if err := f.Truncate(newFileSize); err != nil {
			return fmt.Errorf("failed to extend file: %w", err)
		}
	}

	if ilstAtom != nil {
		oldSize := calculateAtomSize(int(ilstAtom.DataSize))
		newSize := calculateAtomSize(len(metadataData))
		delta := int(newSize) - int(oldSize)

		if err := shiftDataAfter(f, ilstAtom.Offset+8+int64(oldSize), int64(delta)); err != nil {
			return fmt.Errorf("failed to shift data: %w", err)
		}

		if err := writeILSTAtom(f, ilstAtom.Offset, metadataData); err != nil {
			return fmt.Errorf("failed to write ilst atom: %w", err)
		}
	} else {
		moovAtom := findMoovAtom(atoms)
		if moovAtom == nil {
			return errors.New("missing moov atom")
		}

		insertionOffset := moovAtom.Offset + 8
		if err := shiftDataAfter(f, insertionOffset, int64(calculateAtomSize(len(metadataData))+8)); err != nil {
			return fmt.Errorf("failed to shift data for new atom: %w", err)
		}

		newILSTOffset := insertionOffset
		if err := writeFullAtom(f, newILSTOffset, []byte("ilst"), metadataData); err != nil {
			return fmt.Errorf("failed to write new ilst atom: %w", err)
		}
	}

	updateFileTimes(filePath)

	return nil
}

type mp4Atom struct {
	Type     [4]byte
	Size     uint32
	DataSize uint32
	Offset   int64
	Children []mp4Atom
}

func parseMP4Atoms(f *os.File) ([]mp4Atom, error) {
	var atoms []mp4Atom
	offset := int64(0)

	for {
		header := make([]byte, 8)
		n, err := f.ReadAt(header, offset)
		if err != nil || n < 8 {
			break
		}

		size := binary.BigEndian.Uint32(header[:4])
		atomType := [4]byte{}
		copy(atomType[:], header[4:8])

		if size == 0 {
			break
		}

		if size == 1 {
			extendedSize := make([]byte, 8)
			if _, err := f.ReadAt(extendedSize, offset+8); err != nil || len(extendedSize) < 8 {
				break
			}
			size = binary.BigEndian.Uint32(extendedSize[4:8])
		}

		var dataSize uint32
		if size >= 8 {
			dataSize = size - 8
		}

		atom := mp4Atom{
			Type:     atomType,
			Size:     size,
			DataSize: dataSize,
			Offset:   offset,
		}

		if isContainerAtom(atomType) {
			childOffset := offset + 8
			childEnd := offset + int64(size)
			for childOffset < childEnd {
				childHeader := make([]byte, 8)
				m, err := f.ReadAt(childHeader, childOffset)
				if err != nil || m < 8 {
					break
				}
				childSize := binary.BigEndian.Uint32(childHeader[:4])
				if childSize == 0 {
					break
				}
				childType := [4]byte{}
				copy(childType[:], childHeader[4:8])

				if isContainerAtom(childType) {
					childAtoms, err := parseContainerAtom(f, childOffset)
					if err == nil {
						atom.Children = append(atom.Children, childAtoms...)
					}
				} else {
					childAtom := mp4Atom{
						Type:     childType,
						Size:     childSize,
						DataSize: childSize - 8,
						Offset:   childOffset,
					}
					atom.Children = append(atom.Children, childAtom)
				}

				childOffset += int64(childSize)
			}
		}

		atoms = append(atoms, atom)
		offset += int64(size)
	}

	return atoms, nil
}

func parseContainerAtom(f *os.File, offset int64) ([]mp4Atom, error) {
	var atoms []mp4Atom

	header := make([]byte, 8)
	if _, err := f.ReadAt(header, offset); err != nil || len(header) < 8 {
		return nil, err
	}

	parentSize := binary.BigEndian.Uint32(header[:4])
	childEnd := offset + int64(parentSize) - 8

	childOffset := offset + 8
	for childOffset < childEnd {
		childHeader := make([]byte, 8)
		n, err := f.ReadAt(childHeader, childOffset)
		if err != nil || n < 8 {
			break
		}
		childSize := binary.BigEndian.Uint32(childHeader[:4])
		if childSize == 0 {
			break
		}
		childType := [4]byte{}
		copy(childType[:], childHeader[4:8])

		atom := mp4Atom{
			Type:     childType,
			Size:     childSize,
			DataSize: childSize - 8,
			Offset:   childOffset,
		}

		atoms = append(atoms, atom)
		childOffset += int64(childSize)
	}

	return atoms, nil
}

func isContainerAtom(atomType [4]byte) bool {
	containerTypes := map[string]bool{
		"moov": true,
		"trak": true,
		"mdia": true,
		"minf": true,
		"dinf": true,
		"stbl": true,
		"udta": true,
		"ilst": true,
		"meta": true,
		"hdlr": true,
	}
	return containerTypes[string(atomType[:])]
}

func findMoovAtom(atoms []mp4Atom) *mp4Atom {
	for i := range atoms {
		if bytes.Equal(atoms[i].Type[:], []byte("moov")) {
			return &atoms[i]
		}
	}
	return nil
}

func findILSTAtom(atoms []mp4Atom) *mp4Atom {
	for i := range atoms {
		if bytes.Equal(atoms[i].Type[:], []byte("ilst")) {
			return &atoms[i]
		}
		if len(atoms[i].Children) > 0 {
			if child := findILSTAtom(atoms[i].Children); child != nil {
				return child
			}
		}
	}
	return nil
}

func encodeILSTMetadata(tags Tags) ([]byte, error) {
	data := bytes.NewBuffer(nil)

	metadataPairs := map[string]string{
		"\xa9nam": TagTitle,
		"\xa9ART": TagArtist,
		"\xa9alb": TagAlbum,
		"\xa2A2":  TagAlbumArtist,
		"\xa9day": TagYear,
		"\xa9gen": TagGenre,
		"trkn":   TagTrackNumber,
		"disk":   TagDiscNumber,
		"cnmt":   TagComment,
	}

	order := []string{"\xa9nam", "\xa9ART", "\xa9alb", "\xa2A2", "\xa9day", "\xa9gen", "trkn", "disk", "cnmt"}

	for _, key := range order {
		tagKey := metadataPairs[key]
		if value, ok := tags[tagKey]; ok && value != "" {
			atomData := encodeMP4Value(key, value, tagKey)
			if len(atomData) > 0 {
				data.Write(atomData)
			}
		}
	}

	if data.Len() == 0 {
		return nil, nil
	}

	return data.Bytes(), nil
}

func encodeMP4Value(atomType, value, tagKey string) []byte {
	var data []byte

	switch tagKey {
	case TagTrackNumber, TagDiscNumber:
		data = encodeIntegerList(value, atomType)
	case TagComment:
		data = encodeUTF8Text(value, atomType)
	default:
		data = encodeUTF8Text(value, atomType)
	}

	if len(data) == 0 {
		return nil
	}

	atomSize := uint32(len(data)) + 8

	atom := make([]byte, 8)
	binary.BigEndian.PutUint32(atom[0:4], atomSize)
	copy(atom[4:8], []byte(atomType))

	return append(atom, data...)
}

func encodeUTF8Text(value, atomType string) []byte {
	data := bytes.NewBuffer(nil)

	locale := []byte{0x00, 0x65, 0x6E, 0x67}

	switch atomType {
	case "\xa9nam", "\xa9ART", "\xa9alb", "\xa2A2", "\xa9day", "\xa9gen":
		data.Write(locale)
		data.WriteString(value)
		data.WriteByte(0x00)
	default:
		data.Write(locale)
		data.WriteString(value)
		data.WriteByte(0x00)
	}

	return data.Bytes()
}

func encodeIntegerList(value, atomType string) []byte {
	data := bytes.NewBuffer(nil)

	var num, total int
	fmt.Sscanf(value, "%d/%d", &num, &total)
	if total == 0 {
		num, _ = strconv.Atoi(value)
	}

	atomData := make([]byte, 4)
	atomData[0] = 0x00
	binary.BigEndian.PutUint16(atomData[2:], uint16(num))

	data.Write(atomData)

	if total > 0 {
		totalData := make([]byte, 4)
		totalData[0] = 0x00
		binary.BigEndian.PutUint16(totalData[2:], uint16(total))
		data.Write(totalData)
	}

	return data.Bytes()
}

func calculateAtomSize(dataSize int) int {
	return dataSize + 8
}

func writeFullAtom(f *os.File, offset int64, atomType []byte, data []byte) error {
	atomSize := uint32(len(data)) + 8

	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], atomSize)
	copy(header[4:8], atomType)

	if _, err := f.WriteAt(header, offset); err != nil {
		return err
	}
	if _, err := f.WriteAt(data, offset+8); err != nil {
		return err
	}

	return nil
}

func writeILSTAtom(f *os.File, offset int64, data []byte) error {
	return writeFullAtom(f, offset, []byte("ilst"), data)
}

func shiftDataAfter(f *os.File, position int64, delta int64) error {
	if delta <= 0 {
		return nil
	}

	fileSize, err := f.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}

	buf := make([]byte, 8192)
	for offset := fileSize; offset > position; offset -= int64(len(buf)) {
		if offset < position+int64(len(buf)) {
			buf = buf[:offset-position]
			offset = position
		}

		dest := offset + delta
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			return err
		}

		_, err = f.WriteAt(buf, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateFileTimes(filePath string) error {
	now := time.Now()
	return os.Chtimes(filePath, now, now)
}

func init() {
	_ = os.Stdin
}