package tests

import "github.com/navidrome/navidrome/model"

type MockTranscodingRepo struct {
	model.TranscodingRepository
}

func (m *MockTranscodingRepo) Get(id string) (*model.Transcoding, error) {
	return &model.Transcoding{ID: id, TargetFormat: "mp3", DefaultBitRate: 160}, nil
}

func (m *MockTranscodingRepo) FindByFormat(format string) (*model.Transcoding, error) {
	switch format {
	case "mp3":
		return &model.Transcoding{ID: "mp31", TargetFormat: "mp3", DefaultBitRate: 160}, nil
	case "oga":
		return &model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 128}, nil
	case "opus":
		return &model.Transcoding{ID: "opus1", TargetFormat: "opus", DefaultBitRate: 96}, nil
	case "flac":
		return &model.Transcoding{ID: "flac1", TargetFormat: "flac", DefaultBitRate: 0, Command: "ffmpeg -i %s -ss %t -map 0:a:0 -v 0 -c:a flac -f flac -"}, nil
	case "aac":
		return &model.Transcoding{ID: "aac1", TargetFormat: "aac", DefaultBitRate: 256, Command: "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -"}, nil
	default:
		return nil, model.ErrNotFound
	}
}
