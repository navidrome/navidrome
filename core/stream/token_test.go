package stream

import (
	"context"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	var (
		ds  *tests.MockDataStore
		ff  *tests.MockFFmpeg
		svc TranscodeDecider
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{
			MockedProperty:    &tests.MockedPropertyRepo{},
			MockedTranscoding: &tests.MockTranscodingRepo{},
		}
		ff = tests.NewMockFFmpeg("")
		auth.Init(ds)
		svc = NewTranscodeDecider(ds, ff)
	})

	Describe("Token round-trip", func() {
		var (
			sourceTime time.Time
			impl       *deciderService
		)

		BeforeEach(func() {
			sourceTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
			impl = svc.(*deciderService)
		})

		It("creates and parses a direct play token", func() {
			decision := &TranscodeDecision{
				MediaID:         "media-123",
				CanDirectPlay:   true,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-123"))
			Expect(params.DirectPlay).To(BeTrue())
			Expect(params.TargetFormat).To(BeEmpty())
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(sourceTime.Unix()))
		})

		It("creates and parses a transcode token with kbps bitrate", func() {
			decision := &TranscodeDecision{
				MediaID:         "media-456",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "mp3",
				TargetBitrate:   256, // kbps
				TargetChannels:  2,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-456"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("mp3"))
			Expect(params.TargetBitrate).To(Equal(256)) // kbps
			Expect(params.TargetChannels).To(Equal(2))
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(sourceTime.Unix()))
		})

		It("creates and parses a transcode token with sample rate", func() {
			decision := &TranscodeDecision{
				MediaID:          "media-789",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "flac",
				TargetBitrate:    0,
				TargetChannels:   2,
				TargetSampleRate: 48000,
				SourceUpdatedAt:  sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-789"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("flac"))
			Expect(params.TargetSampleRate).To(Equal(48000))
			Expect(params.TargetChannels).To(Equal(2))
		})

		It("creates and parses a transcode token with bit depth", func() {
			decision := &TranscodeDecision{
				MediaID:         "media-bd",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "flac",
				TargetBitrate:   0,
				TargetChannels:  2,
				TargetBitDepth:  24,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-bd"))
			Expect(params.TargetBitDepth).To(Equal(24))
		})

		It("omits bit depth from token when 0", func() {
			decision := &TranscodeDecision{
				MediaID:         "media-nobd",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "mp3",
				TargetBitrate:   256,
				TargetBitDepth:  0,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.TargetBitDepth).To(Equal(0))
		})

		It("omits sample rate from token when 0", func() {
			decision := &TranscodeDecision{
				MediaID:          "media-100",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "mp3",
				TargetBitrate:    256,
				TargetSampleRate: 0,
				SourceUpdatedAt:  sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.TargetSampleRate).To(Equal(0))
		})

		It("truncates SourceUpdatedAt to seconds", func() {
			timeWithNanos := time.Date(2025, 6, 15, 10, 30, 0, 123456789, time.UTC)
			decision := &TranscodeDecision{
				MediaID:         "media-trunc",
				CanDirectPlay:   true,
				SourceUpdatedAt: timeWithNanos,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(timeWithNanos.Truncate(time.Second).Unix()))
		})

		It("rejects an invalid token", func() {
			_, err := impl.parseTranscodeParams("invalid-token")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ResolveRequestFromToken", func() {
		var sourceTime time.Time

		BeforeEach(func() {
			sourceTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
		})

		createTokenForMedia := func(mediaID string, updatedAt time.Time) string {
			decision := &TranscodeDecision{
				MediaID:         mediaID,
				CanDirectPlay:   true,
				SourceUpdatedAt: updatedAt,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())
			return token
		}

		It("returns stream request for valid token", func() {
			mf := &model.MediaFile{ID: "song-1", UpdatedAt: sourceTime}
			token := createTokenForMedia("song-1", sourceTime)

			req, err := svc.ResolveRequestFromToken(ctx, token, mf, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(req.Format).To(BeEmpty()) // direct play has no target format
		})

		It("returns ErrTokenInvalid for invalid token", func() {
			mf := &model.MediaFile{ID: "song-1", UpdatedAt: sourceTime}
			_, err := svc.ResolveRequestFromToken(ctx, "bad-token", mf, 0)
			Expect(err).To(MatchError(ContainSubstring(ErrTokenInvalid.Error())))
		})

		It("returns ErrTokenInvalid when mediaID does not match token", func() {
			mf := &model.MediaFile{ID: "song-2", UpdatedAt: sourceTime}
			token := createTokenForMedia("song-1", sourceTime)

			_, err := svc.ResolveRequestFromToken(ctx, token, mf, 0)
			Expect(err).To(MatchError(ContainSubstring(ErrTokenInvalid.Error())))
		})

		It("returns ErrTokenStale when media file has changed", func() {
			newTime := sourceTime.Add(1 * time.Hour)
			mf := &model.MediaFile{ID: "song-1", UpdatedAt: newTime}
			token := createTokenForMedia("song-1", sourceTime)

			_, err := svc.ResolveRequestFromToken(ctx, token, mf, 0)
			Expect(err).To(MatchError(ErrTokenStale))
		})
	})

	Describe("paramsFromToken", func() {
		It("returns error when media ID is missing", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			token, _, err := tokenAuth.Encode(map[string]any{"ua": int64(1700000000)})
			Expect(err).NotTo(HaveOccurred())

			_, err = paramsFromToken(token)
			Expect(err).To(MatchError(ContainSubstring("missing media ID")))
		})

		It("returns error when source timestamp is missing", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			token, _, err := tokenAuth.Encode(map[string]any{"mid": "song-5"})
			Expect(err).NotTo(HaveOccurred())

			_, err = paramsFromToken(token)
			Expect(err).To(MatchError(ContainSubstring("missing source timestamp")))
		})
	})
})
