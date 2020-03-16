package engine

import "github.com/deluan/navidrome/model"

type mockTranscodingRepository struct {
	model.TranscodingRepository
}

func (m *mockTranscodingRepository) Get(id string) (*model.Transcoding, error) {
	return &model.Transcoding{ID: id, TargetFormat: "mp3", DefaultBitRate: 160}, nil
}

func (m *mockTranscodingRepository) FindByFormat(format string) (*model.Transcoding, error) {
	switch format {
	case "mp3":
		return &model.Transcoding{ID: "mp31", TargetFormat: "mp3", DefaultBitRate: 160}, nil
	case "oga":
		return &model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 128}, nil
	default:
		return nil, model.ErrNotFound
	}
}
