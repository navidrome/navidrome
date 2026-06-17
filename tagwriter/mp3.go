package tagwriter

import (
	"fmt"

	"github.com/bogem/id3v2"
)

func writeMP3Tags(filePath string, tags Tags) error {
	tagFile, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tagFile.Close()

	tagFile.SetDefaultEncoding(id3v2.EncodingUTF8)

	if title, ok := tags[TagTitle]; ok && title != "" {
		tagFile.SetTitle(title)
	}

	if artist, ok := tags[TagArtist]; ok && artist != "" {
		tagFile.SetArtist(artist)
	}

	if album, ok := tags[TagAlbum]; ok && album != "" {
		tagFile.SetAlbum(album)
	}

	if albumArtist, ok := tags[TagAlbumArtist]; ok && albumArtist != "" {
		tagFile.AddTextFrame("TPE1", id3v2.EncodingUTF8, albumArtist)
	}

	if year, ok := tags[TagYear]; ok && year != "" {
		tagFile.SetYear(year)
	}

	if genre, ok := tags[TagGenre]; ok && genre != "" {
		tagFile.SetGenre(genre)
	}

	if trackNum, ok := tags[TagTrackNumber]; ok && trackNum != "" {
		trackTotal, _ := tags[TagTrackTotal]
		trackFrame := fmt.Sprintf("%s/%s", trackNum, trackTotal)
		tagFile.AddTextFrame("TRCK", id3v2.EncodingUTF8, trackFrame)
	}

	if discNum, ok := tags[TagDiscNumber]; ok && discNum != "" {
		discTotal, _ := tags[TagDiscTotal]
		discFrame := fmt.Sprintf("%s/%s", discNum, discTotal)
		tagFile.AddTextFrame("TPOS", id3v2.EncodingUTF8, discFrame)
	}

	if comment, ok := tags[TagComment]; ok && comment != "" {
		tagFile.AddCommentFrame(id3v2.CommentFrame{
			Language:          "eng",
			Description:       "",
			Text:              comment,
			Encoding:          id3v2.EncodingUTF8,
		})
	}

	if err := tagFile.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	return nil
}