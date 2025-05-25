package lyrics

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	var lyricsList model.LyricList
	var err error

	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.LyricsPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			lyricsList, err = fromEmbedded(ctx, mf)
		case strings.HasPrefix(pattern, "."):
			lyricsList, err = fromExternalFile(ctx, mf, pattern)
		default:
			log.Error(ctx, "Invalid lyric pattern", "pattern", pattern)
		}

		if err != nil {
			log.Error(ctx, "error parsing lyrics", "source", pattern, err)
		}

		if len(lyricsList) > 0 {
			return lyricsList, nil
		}
	}

	return nil, nil
}
