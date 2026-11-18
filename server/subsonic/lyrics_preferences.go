package subsonic

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

const (
	lyricsPersonalKeyPrefix        = "lyrics.personal.v1:"
	maxPreferenceWriteAttempts     = 4
	lyricsPersonalSelectionPinned  = "pinned"
	lyricsPersonalSelectionRebound = "rebound"
	lyricsPersonalSelectionPending = "pending"
	lyricsPersonalSelectionGone    = "unavailable"
	lyricsPersonalReasonAmbiguous  = "ambiguous"
	lyricsPersonalReasonMismatch   = "mismatch"
	lyricsPersonalReasonMissing    = "not-available"
)

var (
	errPersonalConflict    = errors.New("lyrics preference conflict")
	errPersonalUnsupported = errors.New("lyrics preference is not supported")
	errPersonalTooLarge    = errors.New("lyrics preference is too large")
	errPersonalFull        = errors.New("lyrics preference list is full")
	errNoPreferenceChange  = errors.New("no lyrics preference change")
)

// LyricsPreferenceUpdate is one validated client correction request.
type LyricsPreferenceUpdate struct {
	TrackID   string
	Action    string
	Selection string
	Local     string
	OffsetMs  int64
	Rejection string
}

func lyricsPreferenceKey(trackID string) string {
	return lyricsPersonalKeyPrefix + trackID
}

// getLyricsBySongIdWithHeaders marks personalized enhanced responses as
// private so no shared cache can replay one listener's choices to another.
func (api *Router) getLyricsBySongIdWithHeaders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if enhanced, _ := req.Params(r).Bool("enhanced"); enhanced {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	return api.GetLyricsBySongId(r)
}

func (api *Router) getLyricsVariantsWithHeaders(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	w.Header().Set("Cache-Control", "private, no-store")
	return api.GetLyricsVariants(r)
}

// SetLyricsPreference applies one deliberate personal action. The user comes
// only from the authenticated request context, and the media file is loaded
// through that context so the existing library access filter applies.
func (api *Router) SetLyricsPreference(r *http.Request) (*responses.Subsonic, error) {
	update, err := parseLyricsPreferenceUpdate(r)
	if err != nil {
		return nil, err
	}
	user, ok := request.UserFrom(r.Context())
	if !ok || user.ID == "" {
		return nil, newError(responses.ErrorAuthorizationFail)
	}
	// Ignore any client-supplied user selector: ownership is the authenticated
	// account only.
	mediaFile, err := api.ds.MediaFile().Get(r.Context(), update.TrackID)
	if err != nil {
		return nil, err
	}
	variants, _, _ := api.lyrics.ListLyricsVariantsResult(r.Context(), mediaFile)
	locals := lyrics.LocalVariants(r.Context(), mediaFile)
	provider, local, err := api.validateLyricsPreferenceUpdate(r.Context(), mediaFile, update, variants, locals)
	if err != nil {
		return nil, err
	}
	record, err := api.mutatePersonal(r.Context(), user.ID, mediaFile.ID, func(record *model.LyricsPreferenceRecord) error {
		return applyLyricsPreferenceUpdate(record, update, provider, local)
	})
	if err != nil {
		return nil, lyricsPreferenceError(err)
	}
	_, personal := applyLyricsPersonal(record, personalEntries(variants, locals), true)
	response := newResponse()
	response.LyricsPersonal = personal
	return response, nil
}

func parseLyricsPreferenceUpdate(r *http.Request) (LyricsPreferenceUpdate, error) {
	params := req.Params(r)
	id, err := params.String("id")
	if err != nil {
		return LyricsPreferenceUpdate{}, err
	}
	action, err := params.String("action")
	if err != nil {
		return LyricsPreferenceUpdate{}, err
	}
	update := LyricsPreferenceUpdate{
		TrackID:   id,
		Action:    strings.ToLower(strings.TrimSpace(action)),
		Selection: strings.TrimSpace(params.StringOr("selection", "")),
		Local:     strings.TrimSpace(params.StringOr("local", "")),
		Rejection: strings.TrimSpace(params.StringOr("rejection", "")),
	}
	if offset, parseErr := params.Int64("offsetMs"); parseErr == nil {
		update.OffsetMs = offset
	} else if update.Action == "offset" {
		return LyricsPreferenceUpdate{}, parseErr
	}
	return update, nil
}

func exactlyOneOf(selection, local string) error {
	if (selection == "") == (local == "") {
		return newError(responses.ErrorGeneric, "invalid lyrics preference parameters")
	}
	return nil
}

func (api *Router) validateLyricsPreferenceUpdate(ctx context.Context, mf *model.MediaFile, update LyricsPreferenceUpdate, variants []model.LyricsVariant, locals []model.LyricsLocalIdentity) (*model.LyricsVariant, *model.LyricsLocalIdentity, error) {
	switch update.Action {
	case "auto":
		if update.Selection != "" || update.Local != "" || update.Rejection != "" {
			return nil, nil, newError(responses.ErrorGeneric, "invalid lyrics preference parameters")
		}
		return nil, nil, nil
	case "reject":
		if update.Selection == "" || update.Local != "" {
			return nil, nil, newError(responses.ErrorGeneric, "rejections apply only to provider recordings")
		}
		variant, err := api.validateProviderSelection(ctx, mf, variants, update.Selection)
		if err != nil {
			return nil, nil, err
		}
		return &variant, nil, nil
	case "restore":
		if update.Rejection == "" || update.Selection != "" || update.Local != "" {
			return nil, nil, newError(responses.ErrorGeneric, "invalid lyrics preference parameters")
		}
		return nil, nil, nil
	case "select", "offset":
		if err := exactlyOneOf(update.Selection, update.Local); err != nil {
			return nil, nil, err
		}
		if update.Selection != "" {
			variant, err := api.validateProviderSelection(ctx, mf, variants, update.Selection)
			if err != nil {
				return nil, nil, err
			}
			if update.Action == "select" && isMismatch(variant.Match) {
				return nil, nil, newError(responses.ErrorGeneric, "lyrics selection is a known mismatch; reject it instead")
			}
			return &variant, nil, nil
		}
		identity, err := validateLocalSelection(locals, update.Local)
		if err != nil {
			return nil, nil, err
		}
		return nil, &identity, nil
	default:
		return nil, nil, newError(responses.ErrorGeneric, "invalid lyrics preference action")
	}
}

func applyLyricsPreferenceUpdate(record *model.LyricsPreferenceRecord, update LyricsPreferenceUpdate, provider *model.LyricsVariant, local *model.LyricsLocalIdentity) error {
	switch update.Action {
	case "auto":
		return clearProviderAndLocalSelection(record)
	case "select":
		return applySelect(record, provider, local)
	case "offset":
		return applyOffset(record, provider, local, update.OffsetMs)
	case "reject":
		return applyRejection(record, provider)
	case "restore":
		return applyRestore(record, update.Rejection)
	}
	return errNoPreferenceChange
}

func (api *Router) validateProviderSelection(ctx context.Context, mf *model.MediaFile, variants []model.LyricsVariant, token string) (model.LyricsVariant, error) {
	plugin, identity, ok := model.DecodeLyricsSelection(token)
	if !ok {
		return model.LyricsVariant{}, newError(responses.ErrorGeneric, "invalid lyrics selection")
	}
	for _, variant := range variants {
		if variant.PluginID == plugin && variant.Identity == identity {
			return variant, nil
		}
	}
	return model.LyricsVariant{}, newError(responses.ErrorDataNotFound, "lyrics selection is not currently available")
}

func validateLocalSelection(locals []model.LyricsLocalIdentity, token string) (model.LyricsLocalIdentity, error) {
	if !strings.HasPrefix(token, "local:v1:") || len(token) > model.MaxLyricsLocalTokenBytes {
		return model.LyricsLocalIdentity{}, newError(responses.ErrorGeneric, "invalid local lyrics selection")
	}
	for _, identity := range locals {
		if identity.Token == token {
			return identity, nil
		}
	}
	return model.LyricsLocalIdentity{}, newError(responses.ErrorDataNotFound, "local lyrics are not currently available")
}

func isMismatch(match *model.LyricsMatch) bool {
	return match != nil && match.Status == model.LyricsMatchMismatch
}

func (api *Router) loadPersonal(ctx context.Context, userID, trackID string) (model.LyricsPreferenceRecord, *string, model.LyricsPreferenceStatus, error) {
	raw, err := api.ds.UserProps().Get(ctx, userID, lyricsPreferenceKey(trackID))
	if errors.Is(err, model.ErrNotFound) {
		return model.LyricsPreferenceRecord{Version: model.LyricsPreferenceVersion}, nil, model.LyricsPreferenceOK, nil
	}
	if err != nil {
		return model.LyricsPreferenceRecord{}, nil, model.LyricsPreferenceOK, err
	}
	record, status := model.DecodeLyricsPreference(raw)
	return record, &raw, status, nil
}

// mutatePersonal is the atomic read/modify/write boundary. The callback applies
// only the requested action to the latest record; a compare-and-swap against
// the exact value read retries with fresh state when another writer won, so
// concurrent disjoint updates are merged instead of lost.
func (api *Router) mutatePersonal(ctx context.Context, userID, trackID string, mutate func(*model.LyricsPreferenceRecord) error) (model.LyricsPreferenceRecord, error) {
	for range maxPreferenceWriteAttempts {
		record, raw, status, err := api.loadPersonal(ctx, userID, trackID)
		if err != nil {
			return model.LyricsPreferenceRecord{}, err
		}
		if status != model.LyricsPreferenceOK {
			return model.LyricsPreferenceRecord{}, errPersonalUnsupported
		}
		beforeRevision := record.Revision
		if err := mutate(&record); err != nil {
			if errors.Is(err, errNoPreferenceChange) {
				return record, nil
			}
			return model.LyricsPreferenceRecord{}, err
		}
		if raw == nil && beforeRevision == 0 && record.Revision == 0 &&
			record.Selection == nil && len(record.Offsets) == 0 && len(record.Rejections) == 0 {
			return record, nil
		}
		record.Version = model.LyricsPreferenceVersion
		record.Revision = beforeRevision + 1
		encoded, ok := model.EncodeLyricsPreference(record)
		if !ok {
			return model.LyricsPreferenceRecord{}, errPersonalTooLarge
		}
		swapped, err := api.ds.UserProps().CompareAndSwap(ctx, userID, lyricsPreferenceKey(trackID), encoded, raw)
		if err != nil {
			return model.LyricsPreferenceRecord{}, err
		}
		if swapped {
			return record, nil
		}
	}
	return model.LyricsPreferenceRecord{}, errPersonalConflict
}

func clearProviderAndLocalSelection(record *model.LyricsPreferenceRecord) error {
	if record.Selection == nil {
		return errNoPreferenceChange
	}
	record.Selection = nil
	return nil
}

func applySelect(record *model.LyricsPreferenceRecord, provider *model.LyricsVariant, local *model.LyricsLocalIdentity) error {
	if record.Selection != nil {
		if provider != nil && record.Selection.Kind == model.LyricsPreferenceKindProvider &&
			record.Selection.Plugin == provider.PluginID && record.Selection.Identity != nil &&
			*record.Selection.Identity == provider.Identity {
			return errNoPreferenceChange
		}
		if local != nil && record.Selection.Kind == model.LyricsPreferenceKindLocal &&
			record.Selection.Local != nil && record.Selection.Local.SameAddress(*local) {
			return errNoPreferenceChange
		}
	}
	if provider != nil {
		identity := provider.Identity
		record.Selection = &model.LyricsPreferenceSelection{
			Kind:     model.LyricsPreferenceKindProvider,
			Plugin:   provider.PluginID,
			Provider: provider.Provider,
			Identity: &identity,
		}
		return nil
	}
	identity := *local
	record.Selection = &model.LyricsPreferenceSelection{
		Kind:  model.LyricsPreferenceKindLocal,
		Local: &identity,
	}
	return nil
}

func applyOffset(record *model.LyricsPreferenceRecord, provider *model.LyricsVariant, local *model.LyricsLocalIdentity, offsetMs int64) error {
	offsetMs = model.ClampLyricsOffset(offsetMs)
	var key string
	var offset model.LyricsPreferenceOffset
	if provider != nil {
		identity := provider.Identity
		scope, stable := identity.RecordingScope()
		if stable {
			offset = model.LyricsPreferenceOffset{
				Kind: model.LyricsPreferenceKindProvider, Plugin: provider.PluginID,
				Instance: identity.Instance, Recording: scope, Language: identity.Language,
				Role: identity.Role, Format: identity.Format, Stable: true,
			}
		} else {
			offset = model.LyricsPreferenceOffset{
				Kind: model.LyricsPreferenceKindProvider, Plugin: provider.PluginID,
				Instance: identity.Instance, Recording: identity.Recording, Language: identity.Language,
				Role: identity.Role, Format: identity.Format,
			}
		}
		key = offset.Key()
	} else {
		identity := *local
		offset = model.LyricsPreferenceOffset{
			Kind: model.LyricsPreferenceKindLocal, Local: &identity,
		}
		key = "local\x00" + identity.Token
	}
	for i := range record.Offsets {
		if record.Offsets[i].Key() != key {
			continue
		}
		if offsetMs == 0 {
			record.Offsets = append(record.Offsets[:i], record.Offsets[i+1:]...)
			return nil
		}
		if record.Offsets[i].OffsetMs == offsetMs {
			return errNoPreferenceChange
		}
		record.Offsets[i].OffsetMs = offsetMs
		return nil
	}
	if offsetMs == 0 {
		return errNoPreferenceChange
	}
	if len(record.Offsets) >= model.MaxLyricsPreferenceOffsets {
		return errPersonalFull
	}
	offset.OffsetMs = offsetMs
	record.Offsets = append(record.Offsets, offset)
	return nil
}

func applyRejection(record *model.LyricsPreferenceRecord, provider *model.LyricsVariant) error {
	scope, stable := provider.Identity.RecordingScope()
	if !stable {
		return newError(responses.ErrorGeneric, "this source cannot identify the recording reliably, so it cannot be rejected")
	}
	for _, rejection := range record.Rejections {
		if rejection.Plugin == provider.PluginID && rejection.Instance == provider.Identity.Instance && rejection.Recording == scope {
			return errNoPreferenceChange
		}
	}
	if len(record.Rejections) >= model.MaxLyricsPreferenceRejections {
		return errPersonalFull
	}
	record.Rejections = append(record.Rejections, model.LyricsPreferenceRejected{
		Plugin: provider.PluginID, Provider: provider.Provider,
		Instance: provider.Identity.Instance, Recording: scope,
	})
	if record.Selection != nil && record.Selection.Kind == model.LyricsPreferenceKindProvider &&
		record.Selection.Identity != nil && record.Selection.Identity.Instance == provider.Identity.Instance {
		if selectionScope, ok := record.Selection.Identity.RecordingScope(); ok && selectionScope == scope {
			record.Selection = nil
		}
	}
	kept := record.Offsets[:0]
	for _, offset := range record.Offsets {
		if offset.Kind == model.LyricsPreferenceKindProvider &&
			offset.Instance == provider.Identity.Instance && offset.Stable && offset.Recording == scope {
			continue
		}
		kept = append(kept, offset)
	}
	record.Offsets = kept
	return nil
}

func applyRestore(record *model.LyricsPreferenceRecord, token string) error {
	plugin, instance, recording, ok := model.DecodeLyricsRejection(token)
	if !ok {
		return newError(responses.ErrorGeneric, "invalid lyrics rejection")
	}
	kept := record.Rejections[:0]
	removed := false
	for _, rejection := range record.Rejections {
		if rejection.Plugin == plugin && rejection.Instance == instance && rejection.Recording == recording {
			removed = true
			continue
		}
		kept = append(kept, rejection)
	}
	record.Rejections = kept
	if !removed {
		return errNoPreferenceChange
	}
	return nil
}

func lyricsPreferenceError(err error) error {
	var subErr subError
	if errors.As(err, &subErr) {
		return err
	}
	switch {
	case errors.Is(err, errPersonalConflict):
		return newError(responses.ErrorGeneric, "could not save the lyrics preference; try again")
	case errors.Is(err, errPersonalUnsupported):
		return newError(responses.ErrorGeneric, "saved lyrics preferences could not be read by this server version")
	case errors.Is(err, errPersonalTooLarge):
		return newError(responses.ErrorGeneric, "the saved lyrics preference is too large")
	case errors.Is(err, errPersonalFull):
		return newError(responses.ErrorGeneric, "too many saved lyrics corrections for this track")
	default:
		return err
	}
}

// personalEntries builds the lightweight entries the resolver needs from a
// variant listing and the readable local variants. It is used for update
// responses; the read path resolves against the real lyric entries.
func personalEntries(variants []model.LyricsVariant, locals []model.LyricsLocalIdentity) model.LyricList {
	entries := make(model.LyricList, 0, len(variants)+len(locals))
	for _, variant := range variants {
		identity := variant.Identity
		source := &model.LyricsSource{
			Type: model.LyricsSourcePlugin, PluginID: variant.PluginID,
			Provider: variant.Provider, Format: variant.Format,
		}
		entries = append(entries, model.Lyrics{
			Source: source, Variant: &identity, Selection: variant.Selection, Match: variant.Match,
		})
	}
	for _, local := range locals {
		identity := local
		kind := model.LyricsSourceSidecar
		if local.Kind == model.LyricsLocalKindEmbedded {
			kind = model.LyricsSourceEmbedded
		}
		entries = append(entries, model.Lyrics{
			Source: &model.LyricsSource{Type: kind, Format: local.Format}, Local: &identity,
		})
	}
	return entries
}

// personalStateForTrack loads the authenticated listener's personal state and
// resolves it against a real enhanced lyric list.
func (api *Router) personalStateForTrack(ctx context.Context, userID string, mf *model.MediaFile, entries model.LyricList, allSources bool) (model.LyricList, *responses.LyricsPersonal) {
	if userID == "" || mf == nil || mf.ID == "" {
		return entries, nil
	}
	record, _, status, err := api.loadPersonal(ctx, userID, mf.ID)
	if err != nil {
		// Durable state is unavailable: serve the shared lyrics and let the
		// client keep its non-personal behavior rather than guessing.
		return entries, nil
	}
	if status != model.LyricsPreferenceOK {
		return entries, &responses.LyricsPersonal{Unsupported: true}
	}
	return applyLyricsPersonal(record, entries, allSources)
}

func (api *Router) applyPersonalForRequest(ctx context.Context, mf *model.MediaFile, entries model.LyricList, allSources bool) (model.LyricList, *responses.LyricsPersonal) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return entries, nil
	}
	return api.personalStateForTrack(ctx, user.ID, mf, entries, allSources)
}

// filterRejectedVariants removes the listener's rejected provider recordings
// from a variant listing. It is the same personal boundary as playback.
func (api *Router) filterRejectedVariants(ctx context.Context, userID, trackID string, variants []model.LyricsVariant) []model.LyricsVariant {
	if userID == "" || trackID == "" || len(variants) == 0 {
		return variants
	}
	record, _, status, err := api.loadPersonal(ctx, userID, trackID)
	if err != nil || status != model.LyricsPreferenceOK {
		return variants
	}
	return filterRejectedVariantList(record.Rejections, variants)
}

func filterRejectedVariantList(rejections []model.LyricsPreferenceRejected, variants []model.LyricsVariant) []model.LyricsVariant {
	if len(rejections) == 0 {
		return variants
	}
	filtered := make([]model.LyricsVariant, 0, len(variants))
	for _, variant := range variants {
		if rejectionMatchesVariant(rejections, variant) {
			continue
		}
		filtered = append(filtered, variant)
	}
	return filtered
}

func rejectionMatchesVariant(rejections []model.LyricsPreferenceRejected, variant model.LyricsVariant) bool {
	scope, stable := variant.Identity.RecordingScope()
	if !stable {
		return false
	}
	for _, rejection := range rejections {
		if rejection.Plugin == variant.PluginID && rejection.Instance == variant.Identity.Instance && rejection.Recording == scope {
			return true
		}
	}
	return false
}

// applyLyricsPersonal filters rejected provider recordings and resolves the
// stored selection and offsets against the current entries. It never rewrites
// lyric order or timing: pins are an explicit client instruction and offsets
// are metadata for the listener.
func applyLyricsPersonal(record model.LyricsPreferenceRecord, entries model.LyricList, allSources bool) (model.LyricList, *responses.LyricsPersonal) {
	personal := &responses.LyricsPersonal{Revision: record.Revision}
	filtered := make(model.LyricList, 0, len(entries))
	for _, lyric := range entries {
		if rejectionMatches(record.Rejections, lyric) {
			continue
		}
		filtered = append(filtered, lyric)
	}
	personal.Selection = resolveSelection(record.Selection, filtered, allSources)
	personal.Offsets = resolveOffsets(record.Offsets, filtered)
	personal.Rejections = resolveRejections(record.Rejections)
	return filtered, personal
}

func rejectionMatches(rejections []model.LyricsPreferenceRejected, lyric model.Lyrics) bool {
	if len(rejections) == 0 || lyric.Source == nil || lyric.Variant == nil || lyric.Source.PluginID == "" {
		return false
	}
	scope, stable := lyric.Variant.RecordingScope()
	if !stable {
		return false
	}
	for _, rejection := range rejections {
		if rejection.Plugin == lyric.Source.PluginID &&
			rejection.Instance == lyric.Variant.Instance && rejection.Recording == scope {
			return true
		}
	}
	return false
}

func resolveSelection(selection *model.LyricsPreferenceSelection, entries model.LyricList, allSources bool) *responses.LyricsPersonalSelection {
	if selection == nil {
		return nil
	}
	if selection.Kind == model.LyricsPreferenceKindLocal && selection.Local != nil {
		for _, lyric := range entries {
			if lyric.Local != nil && lyric.Local.SameAddress(*selection.Local) {
				identity := *lyric.Local
				return &responses.LyricsPersonalSelection{
					Kind: model.LyricsPreferenceKindLocal, Local: localIdentityResponse(identity),
					Status: lyricsPersonalSelectionPinned,
				}
			}
		}
		identity := *selection.Local
		return &responses.LyricsPersonalSelection{
			Kind: model.LyricsPreferenceKindLocal, Local: localIdentityResponse(identity),
			Status: lyricsPersonalSelectionGone, Reason: lyricsPersonalReasonMissing,
		}
	}
	if selection.Kind != model.LyricsPreferenceKindProvider || selection.Identity == nil {
		return nil
	}
	stored := *selection.Identity
	for _, lyric := range entries {
		if !providerEntryMatches(lyric, selection.Plugin) || lyric.Variant == nil {
			continue
		}
		if *lyric.Variant == stored {
			if isMismatch(lyric.Match) {
				return providerSelectionResponse(selection, lyric, lyricsPersonalSelectionGone, lyricsPersonalReasonMismatch)
			}
			return providerSelectionResponse(selection, lyric, lyricsPersonalSelectionPinned, "")
		}
	}
	scope, stable := stored.RecordingScope()
	if stable {
		var match *model.Lyrics
		distinct := map[string]bool{}
		for i := range entries {
			lyric := &entries[i]
			if !providerEntryMatches(*lyric, selection.Plugin) || lyric.Variant == nil {
				continue
			}
			candidateScope, candidateStable := lyric.Variant.RecordingScope()
			if !candidateStable || candidateScope != scope ||
				lyric.Variant.Instance != stored.Instance ||
				lyric.Variant.Language != stored.Language ||
				lyric.Variant.Role != stored.Role ||
				lyric.Variant.Format != stored.Format {
				continue
			}
			distinct[lyric.Variant.Key()] = true
			match = lyric
		}
		if len(distinct) == 1 && match != nil && !isMismatch(match.Match) {
			return providerSelectionResponse(selection, *match, lyricsPersonalSelectionRebound, "")
		}
		if len(distinct) > 1 {
			return providerSelectionResponse(selection, model.Lyrics{}, lyricsPersonalSelectionGone, lyricsPersonalReasonAmbiguous)
		}
	}
	if !allSources {
		return providerSelectionResponse(selection, model.Lyrics{}, lyricsPersonalSelectionPending, "")
	}
	return providerSelectionResponse(selection, model.Lyrics{}, lyricsPersonalSelectionGone, lyricsPersonalReasonMissing)
}

func providerEntryMatches(lyric model.Lyrics, plugin string) bool {
	return lyric.Source != nil && lyric.Source.PluginID != "" && lyric.Source.PluginID == plugin
}

func providerSelectionResponse(selection *model.LyricsPreferenceSelection, lyric model.Lyrics, status, reason string) *responses.LyricsPersonalSelection {
	identity := *selection.Identity
	response := &responses.LyricsPersonalSelection{
		Kind: model.LyricsPreferenceKindProvider, Plugin: selection.Plugin,
		Provider: selection.Provider, Identity: variantIdentityResponse(identity),
		Status: status, Reason: reason,
	}
	if lyric.Variant != nil {
		identity = *lyric.Variant
		response.Identity = variantIdentityResponse(identity)
		response.Selection = lyric.Selection
		if lyric.Source != nil && lyric.Source.Provider != "" {
			response.Provider = lyric.Source.Provider
		}
	}
	return response
}

func resolveOffsets(offsets []model.LyricsPreferenceOffset, entries model.LyricList) []responses.LyricsPersonalOffset {
	result := make([]responses.LyricsPersonalOffset, 0, len(offsets))
	for _, offset := range offsets {
		if offset.Kind == model.LyricsPreferenceKindLocal && offset.Local != nil {
			for _, lyric := range entries {
				if lyric.Local != nil && lyric.Local.SameAddress(*offset.Local) {
					identity := *lyric.Local
					result = append(result, responses.LyricsPersonalOffset{
						Kind: model.LyricsPreferenceKindLocal, Local: localIdentityResponse(identity),
						OffsetMs: offset.OffsetMs,
					})
					break
				}
			}
			continue
		}
		var match *model.Lyrics
		distinct := map[string]bool{}
		for i := range entries {
			lyric := &entries[i]
			if !providerEntryMatches(*lyric, offset.Plugin) || lyric.Variant == nil || isMismatch(lyric.Match) {
				continue
			}
			identity := *lyric.Variant
			if identity.Language != offset.Language || identity.Role != offset.Role || identity.Format != offset.Format {
				continue
			}
			if offset.Stable {
				scope, stable := identity.RecordingScope()
				if !stable || scope != offset.Recording {
					continue
				}
			} else if identity.Recording != offset.Recording {
				continue
			}
			if identity.Instance != offset.Instance {
				continue
			}
			distinct[identity.Key()] = true
			match = lyric
		}
		if len(distinct) == 1 && match != nil && match.Selection != "" {
			result = append(result, responses.LyricsPersonalOffset{
				Kind: model.LyricsPreferenceKindProvider, Selection: match.Selection,
				OffsetMs: offset.OffsetMs,
			})
		}
	}
	return result
}

func resolveRejections(rejections []model.LyricsPreferenceRejected) []responses.LyricsPersonalRejected {
	result := make([]responses.LyricsPersonalRejected, 0, len(rejections))
	for _, rejection := range rejections {
		token, ok := model.EncodeLyricsRejection(rejection.Plugin, rejection.Instance, rejection.Recording)
		if !ok {
			continue
		}
		result = append(result, responses.LyricsPersonalRejected{
			Token: token, Plugin: rejection.Plugin, Provider: rejection.Provider,
			Instance: rejection.Instance, Recording: rejection.Recording,
		})
	}
	return result
}

func variantIdentityResponse(identity model.LyricsVariantIdentity) *responses.LyricsVariantIdentity {
	return &responses.LyricsVariantIdentity{
		Version: identity.Version, Instance: identity.Instance, Recording: identity.Recording,
		Language: identity.Language, Role: identity.Role, Format: identity.Format,
	}
}

func localIdentityResponse(identity model.LyricsLocalIdentity) *responses.LyricsLocalIdentity {
	return &responses.LyricsLocalIdentity{
		Version: identity.Version, Token: identity.Token, Kind: identity.Kind,
		Format: identity.Format, Language: identity.Language, Role: identity.Role,
	}
}
