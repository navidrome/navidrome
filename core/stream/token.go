package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const tokenTTL = 48 * time.Hour

// params contains the parameters extracted from a transcode token.
// TargetBitrate is in kilobits per second (kbps).
type params struct {
	MediaID          string
	DirectPlay       bool
	TargetFormat     string
	TargetBitrate    int
	TargetChannels   int
	TargetSampleRate int
	TargetBitDepth   int
	SourceUpdatedAt  time.Time
}

// toClaimsMap converts a Decision into a JWT claims map for token encoding.
// Only non-zero transcode fields are included.
func (d *TranscodeDecision) toClaimsMap() map[string]any {
	m := map[string]any{
		"mid":             d.MediaID,
		"ua":              d.SourceUpdatedAt.Truncate(time.Second).Unix(),
		jwt.ExpirationKey: time.Now().Add(tokenTTL).UTC().Unix(),
	}
	if d.CanDirectPlay {
		m["dp"] = true
	}
	if d.CanTranscode && d.TargetFormat != "" {
		m["f"] = d.TargetFormat
		if d.TargetBitrate != 0 {
			m["b"] = d.TargetBitrate
		}
		if d.TargetChannels != 0 {
			m["ch"] = d.TargetChannels
		}
		if d.TargetSampleRate != 0 {
			m["sr"] = d.TargetSampleRate
		}
		if d.TargetBitDepth != 0 {
			m["bd"] = d.TargetBitDepth
		}
	}
	return m
}

// paramsFromToken extracts and validates Params from a parsed JWT token.
// Returns an error if required claims (media ID, source timestamp) are missing.
func paramsFromToken(token jwt.Token) (*params, error) {
	var p params
	var mid string
	if err := token.Get("mid", &mid); err == nil {
		p.MediaID = mid
	}
	if p.MediaID == "" {
		return nil, fmt.Errorf("%w: missing media ID", ErrTokenInvalid)
	}

	var dp bool
	if err := token.Get("dp", &dp); err == nil {
		p.DirectPlay = dp
	}

	ua := getIntClaim(token, "ua")
	if ua != 0 {
		p.SourceUpdatedAt = time.Unix(int64(ua), 0)
	}
	if p.SourceUpdatedAt.IsZero() {
		return nil, fmt.Errorf("%w: missing source timestamp", ErrTokenInvalid)
	}

	var f string
	if err := token.Get("f", &f); err == nil {
		p.TargetFormat = f
	}
	p.TargetBitrate = getIntClaim(token, "b")
	p.TargetChannels = getIntClaim(token, "ch")
	p.TargetSampleRate = getIntClaim(token, "sr")
	p.TargetBitDepth = getIntClaim(token, "bd")
	return &p, nil
}

// getIntClaim extracts an int claim from a JWT token, handling the case where
// the value may be stored as int64 or float64 (common in JSON-based JWT libraries).
func getIntClaim(token jwt.Token, key string) int {
	var v int
	if err := token.Get(key, &v); err == nil {
		return v
	}
	var v64 int64
	if err := token.Get(key, &v64); err == nil {
		return int(v64)
	}
	var f float64
	if err := token.Get(key, &f); err == nil {
		return int(f)
	}
	return 0
}

func (s *deciderService) CreateTranscodeParams(decision *TranscodeDecision) (string, error) {
	return auth.EncodeToken(decision.toClaimsMap())
}

func (s *deciderService) parseTranscodeParams(tokenStr string) (*params, error) {
	token, err := auth.DecodeAndVerifyToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return paramsFromToken(token)
}

func (s *deciderService) ResolveRequestFromToken(ctx context.Context, token string, mf *model.MediaFile, offset int) (Request, error) {
	p, err := s.parseTranscodeParams(token)
	if err != nil {
		return Request{}, errors.Join(ErrTokenInvalid, err)
	}
	if p.MediaID != mf.ID {
		return Request{}, fmt.Errorf("%w: token mediaID %q does not match %q", ErrTokenInvalid, p.MediaID, mf.ID)
	}
	if !mf.UpdatedAt.Truncate(time.Second).Equal(p.SourceUpdatedAt) {
		log.Info(ctx, "Transcode token is stale", "mediaID", mf.ID,
			"tokenUpdatedAt", p.SourceUpdatedAt, "fileUpdatedAt", mf.UpdatedAt)
		return Request{}, ErrTokenStale
	}

	req := Request{Offset: offset}
	if !p.DirectPlay && p.TargetFormat != "" {
		req.Format = p.TargetFormat
		req.BitRate = p.TargetBitrate
		req.SampleRate = p.TargetSampleRate
		req.BitDepth = p.TargetBitDepth
		req.Channels = p.TargetChannels
	}
	return req, nil
}
