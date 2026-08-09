package model

// Feature keys for admin-gated fork features. Absence of an explicit row in
// user_feature_permission means enabled - this is an opt-out model, so
// existing users are unaffected until an admin explicitly revokes access.
//
// AI Genre and AI Mood are deliberately a single "ai_tags" feature, not two:
// both dashboards read the same source=ai aggregate query, split client-side
// by a genre:/mood: tag-name prefix, so there's no backend query shape to
// gate them independently without new query surface.
const (
	FeatureFolders  = "folders"
	FeatureAITags   = "ai_tags"
	FeatureMyTags   = "my_tags"
	FeaturePodcasts = "podcasts"
)

// AllUserFeatures is the list of valid feature keys, used to validate
// SetUserFeaturePermissions input and to fill in defaults for features with
// no explicit row yet.
var AllUserFeatures = []string{FeatureFolders, FeatureAITags, FeatureMyTags, FeaturePodcasts}

// UserFeaturePermission is a per-user, per-feature admin-controlled access grant.
type UserFeaturePermission struct {
	UserID  string `structs:"user_id" json:"userId"`
	Feature string `structs:"feature" json:"feature"`
	Enabled bool   `structs:"enabled" json:"enabled"`
}
