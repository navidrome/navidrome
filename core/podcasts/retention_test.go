package podcasts

import (
	"testing"
	"time"

	"github.com/navidrome/navidrome/model"
)

// episodeAgo builds a test episode published `ago` before now. Ages are
// expressed as durations (not whole days) so tests can keep a comfortable
// margin away from a RetentionDays cutoff boundary - the cutoff is computed
// at call time, a few microseconds after these fixtures are built, so an
// episode aged exactly N*24h would race the boundary and flake.
func episodeAgo(id string, ago time.Duration, size int64) model.PodcastEpisode {
	t := time.Now().Add(-ago)
	return model.PodcastEpisode{ID: id, PublishDate: &t, Size: size}
}

func TestRetentionCandidates(t *testing.T) {
	// episodes are always passed newest-first, matching the "publish_date desc" query.
	episodes := model.PodcastEpisodes{
		episodeAgo("e1", 1*time.Hour, 100),
		episodeAgo("e2", 30*time.Hour, 100),  // ~1.25 days
		episodeAgo("e3", 54*time.Hour, 100),  // ~2.25 days
		episodeAgo("e4", 78*time.Hour, 100),  // ~3.25 days
		episodeAgo("e5", 102*time.Hour, 100), // ~4.25 days
	}

	t.Run("no limits configured", func(t *testing.T) {
		channel := model.PodcastChannel{}
		got := retentionCandidates(channel, episodes)
		if len(got) != 0 {
			t.Fatalf("expected no candidates, got %d", len(got))
		}
	})

	t.Run("retention count keeps only the newest N", func(t *testing.T) {
		channel := model.PodcastChannel{RetentionCount: 2}
		got := retentionCandidates(channel, episodes)
		assertIDs(t, got, []string{"e3", "e4", "e5"})
	})

	t.Run("retention days drops episodes older than the cutoff", func(t *testing.T) {
		channel := model.PodcastChannel{RetentionDays: 2} // cutoff ~48h ago
		got := retentionCandidates(channel, episodes)
		assertIDs(t, got, []string{"e3", "e4", "e5"})
	})

	t.Run("episodes with no publish date are exempt from age-based cleanup", func(t *testing.T) {
		channel := model.PodcastChannel{RetentionDays: 1}
		noDate := model.PodcastEpisode{ID: "nodate", Size: 100}
		got := retentionCandidates(channel, model.PodcastEpisodes{episodes[0], noDate, episodes[4]})
		assertIDs(t, got, []string{"e5"})
	})

	t.Run("max storage keeps newest episodes under budget", func(t *testing.T) {
		sized := model.PodcastEpisodes{
			episodeAgo("s1", 1*time.Hour, 400_000),
			episodeAgo("s2", 2*time.Hour, 400_000),
			episodeAgo("s3", 3*time.Hour, 400_000),
			episodeAgo("s4", 4*time.Hour, 400_000),
		}
		channel := model.PodcastChannel{MaxStorageMB: 1} // 1,048,576 byte budget
		got := retentionCandidates(channel, sized)
		// s1+s2 = 800,000 (under budget); s1+s2+s3 = 1,200,000 (over) -> s3 and everything after are candidates.
		assertIDs(t, got, []string{"s3", "s4"})
	})

	t.Run("zero max storage is unlimited", func(t *testing.T) {
		channel := model.PodcastChannel{MaxStorageMB: 0}
		got := retentionCandidates(channel, episodes)
		if len(got) != 0 {
			t.Fatalf("expected no candidates with MaxStorageMB=0, got %d", len(got))
		}
	})

	t.Run("combined limits union their candidates", func(t *testing.T) {
		channel := model.PodcastChannel{RetentionCount: 3, RetentionDays: 1} // cutoff ~24h ago
		got := retentionCandidates(channel, episodes)
		// count alone would keep e1-e3; days alone (cutoff ~24h) only keeps e1 (1h old).
		// union of both limits keeps just e1.
		assertIDs(t, got, []string{"e2", "e3", "e4", "e5"})
	})
}

func assertIDs(t *testing.T, got model.PodcastEpisodes, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d candidates %v, got %d: %v", len(want), want, len(got), idsOf(got))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("expected candidate %d to be %q, got %q (all: %v)", i, id, got[i].ID, idsOf(got))
		}
	}
}

func idsOf(episodes model.PodcastEpisodes) []string {
	ids := make([]string, len(episodes))
	for i, ep := range episodes {
		ids[i] = ep.ID
	}
	return ids
}
