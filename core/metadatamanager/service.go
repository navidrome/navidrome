package metadatamanager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/navidrome/navidrome/log"
)

type SongRepository interface {
	GetSongPath(ctx context.Context, songID string) (string, error)
	RefreshSong(ctx context.Context, songID string) error
}

type MetadataService interface {
	UpdateTags(ctx context.Context, songID string, tags map[string]string) error
	UpdateArtwork(ctx context.Context, songID string, data io.Reader, mimeType string) error
}

type mp3Service struct {
	repo SongRepository
}

func NewService(repo SongRepository) MetadataService {
	return &mp3Service{repo: repo}
}

func (s *mp3Service) UpdateTags(ctx context.Context, songID string, tags map[string]string) error {
	path, err := s.repo.GetSongPath(ctx, songID)
	if err != nil {
		return fmt.Errorf("could not retrieve song path: %w", err)
	}

	cleanPath := filepath.Clean(path)

	if info, err := os.Stat(cleanPath); err != nil || info.IsDir() {
		return fmt.Errorf("file is inaccessible or is a directory: %s", cleanPath)
	}

	// Ensure we are only processing MP3 files as id3v2 library is specific to ID3 tags
	if !strings.HasSuffix(strings.ToLower(cleanPath), ".mp3") {
		return fmt.Errorf("metadata editing is currently only supported for MP3 files")
	}

	log.Info(ctx, "Updating MP3 tags", "path", cleanPath, "songID", songID)

	tag, err := id3v2.Open(cleanPath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("error opening MP3 file: %w", err)
	}
	defer tag.Close()

	for key, value := range tags {
		switch key {
		case "title":
			tag.SetTitle(value)
		case "artist":
			tag.SetArtist(value)
		case "album":
			tag.SetAlbum(value)
		case "albumArtist":
			tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, value)
		case "genre":
			tag.SetGenre(value)
		case "comment":
			tag.AddCommentFrame(id3v2.CommentFrame{
				Encoding: id3v2.EncodingUTF8,
				Language: "eng",
				Text:     value,
			})
		case "bpm":
			tag.AddTextFrame("TBPM", id3v2.EncodingUTF8, value)
		case "composer":
			tag.AddTextFrame("TCOM", id3v2.EncodingUTF8, value)
		case "grouping":
			tag.AddTextFrame("TIT1", id3v2.EncodingUTF8, value)
		case "mood":
			tag.AddTextFrame("TMOO", id3v2.EncodingUTF8, value)
		case "year":
			tag.SetYear(value)
		case "trackNumber":
			tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, value)
		case "disc":
			tag.AddTextFrame("TPOS", id3v2.EncodingUTF8, value)
		default:
			log.Warn(ctx, "Tag not supported for MP3 metadata. Ignoring.", "tag", key)
		}
	}

	if err = tag.Save(); err != nil {
		return fmt.Errorf("error saving MP3 tags: %w", err)
	}

	// Trigger a rescan of this song so Navidrome updates its database (seems to generate lag)
	return s.repo.RefreshSong(ctx, songID)
}

func (s *mp3Service) UpdateArtwork(ctx context.Context, songID string, data io.Reader, mimeType string) error {
	path, err := s.repo.GetSongPath(ctx, songID)
	if err != nil {
		return err
	}

	cleanPath := filepath.Clean(path)
	if !strings.HasSuffix(strings.ToLower(cleanPath), ".mp3") {
		return fmt.Errorf("artwork embedding is only supported for MP3 files")
	}

	imgBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}

	log.Info(ctx, "Embedding artwork in MP3", "path", cleanPath, "mime", mimeType)

	tag, err := id3v2.Open(cleanPath, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.DeleteFrames("APIC")

	if len(imgBytes) > 0 {
		if mimeType == "" {
			mimeType = "image/jpeg"
		}

		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mimeType,
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     imgBytes,
		})
	}

	if err = tag.Save(); err != nil {
		return err
	}

	return s.repo.RefreshSong(ctx, songID)
}
