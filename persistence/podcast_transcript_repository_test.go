package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastTranscriptRepository", func() {
	var repo model.PodcastTranscriptRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastTranscriptRepository(ctx, GetDBXBuilder())
	})

	Describe("Save and GetByEpisode", func() {
		It("returns saved transcript by episode id", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/t.vtt", MimeType: "text/vtt", Language: "en", Rel: "captions"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].URL).To(Equal("https://example.com/t.vtt"))
			Expect(result[0].MimeType).To(Equal("text/vtt"))
			Expect(result[0].Language).To(Equal("en"))
			Expect(result[0].Rel).To(Equal("captions"))

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())
		})

		It("saves multiple transcripts for one episode", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/t.vtt", MimeType: "text/vtt", Language: "en", Rel: "captions"},
				{EpisodeID: "pe-1", URL: "https://example.com/t.srt", MimeType: "application/x-subrip", Language: "en"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			var mimeTypes []string
			for _, t := range result {
				mimeTypes = append(mimeTypes, t.MimeType)
			}
			Expect(mimeTypes).To(ConsistOf("text/vtt", "application/x-subrip"))

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())
		})

		It("stores empty rel when rel attribute is omitted", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/t.txt", MimeType: "text/plain"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].Rel).To(BeEmpty())

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())
		})

		It("returns empty list for unknown episode", func() {
			result, err := repo.GetByEpisode("no-such-episode")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("GetByEpisodes — bulk query", func() {
		It("returns transcripts for multiple episodes in one query", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/t1.vtt", MimeType: "text/vtt"},
				{EpisodeID: "pe-2", URL: "https://example.com/t2.srt", MimeType: "application/x-subrip"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			result, err := repo.GetByEpisodes([]string{"pe-1", "pe-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			episodeIDs := []string{result[0].EpisodeID, result[1].EpisodeID}
			Expect(episodeIDs).To(ConsistOf("pe-1", "pe-2"))

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())
			Expect(repo.DeleteByEpisode("pe-2")).To(Succeed())
		})

		It("returns empty list for empty id slice", func() {
			result, err := repo.GetByEpisodes([]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("DeleteByEpisode", func() {
		It("deletes only transcripts for the specified episode", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/t1.vtt", MimeType: "text/vtt"},
				{EpisodeID: "pe-2", URL: "https://example.com/t2.vtt", MimeType: "text/vtt"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())

			result1, _ := repo.GetByEpisode("pe-1")
			Expect(result1).To(BeEmpty())

			result2, _ := repo.GetByEpisode("pe-2")
			Expect(result2).To(HaveLen(1))

			Expect(repo.DeleteByEpisode("pe-2")).To(Succeed())
		})

		It("succeeds when deleting transcripts for a non-existent episode", func() {
			Expect(repo.DeleteByEpisode("no-such-episode")).To(Succeed())
		})
	})

	Describe("Save — auto ID generation", func() {
		It("assigns an ID automatically when none is provided", func() {
			transcripts := []model.PodcastTranscript{
				{EpisodeID: "pe-1", URL: "https://example.com/auto.vtt", MimeType: "text/vtt"},
			}
			Expect(repo.Save(transcripts)).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].ID).ToNot(BeEmpty())

			Expect(repo.DeleteByEpisode("pe-1")).To(Succeed())
		})
	})
})
