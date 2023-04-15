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
	default:
		return nil, model.ErrNotFound
	}
}
