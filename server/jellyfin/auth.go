package jellyfin

import (
	"encoding/json"
	"net/http"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

type authenticateByNameRequest struct {
	Username string `json:"Username"`
	Pw       string `json:"Pw"`
}

func (api *Router) authenticateByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body authenticateByNameRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Navidrome stores recoverable passwords; this mirrors Subsonic's validateCredentials plaintext path.
	usr, err := api.ds.User(ctx).FindByUsernameWithPassword(body.Username)
	if body.Pw == "" || err != nil || usr == nil || usr.Password != body.Pw {
		log.Warn(ctx, "Jellyfin API: invalid login", "username", body.Username, "remoteAddr", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Best-effort, like the web UI's validateLogin: without it, Jellyfin-only users show a
	// never/stale "Last Login" in the admin UI.
	if err := api.ds.User(ctx).UpdateLastLoginAt(usr.ID); err != nil {
		log.Error(ctx, "Jellyfin API: could not update last login date", "username", body.Username, err)
	}

	token, err := auth.CreateToken(usr)
	if err != nil {
		api.internalError(w, r, err)
		return
	}

	// SessionInfo is omitted, not partially filled: a stub {Id, UserId} could fail a strict client's
	// parse, and Finamp's login doesn't need it (its AuthenticationResult.sessionInfo is nullable).
	api.ok(w, r, dto.AuthenticationResult{
		User:        userToDto(usr, api.serverName(), api.serverID(ctx)),
		AccessToken: token,
		ServerId:    api.serverID(ctx),
	})
}

// userToDto builds the User object clients expect. Finamp reads Policy and Configuration right after
// login and null-crashes if absent, so both are filled with Navidrome-appropriate defaults.
func userToDto(u *model.User, serverName, serverID string) *dto.UserDto {
	return &dto.UserDto{
		Name:                  u.UserName,
		Id:                    dto.EncodeID(u.ID), // hex like every other id, so lowercased paths stay valid
		ServerId:              serverID,
		ServerName:            serverName,
		HasPassword:           true,
		HasConfiguredPassword: true,
		Policy:                userPolicy(u),
		Configuration:         userConfiguration(),
	}
}

func userPolicy(u *model.User) *dto.UserPolicy {
	return &dto.UserPolicy{
		IsAdministrator:                  u.IsAdmin,
		IsHidden:                         false,
		EnableCollectionManagement:       false,
		EnableSubtitleManagement:         false,
		EnableLyricManagement:            false,
		IsDisabled:                       false,
		BlockedTags:                      []string{},
		AllowedTags:                      []string{},
		EnableUserPreferenceAccess:       true,
		AccessSchedules:                  []string{},
		BlockUnratedItems:                []string{},
		EnableRemoteControlOfOtherUsers:  false,
		EnableSharedDeviceControl:        false,
		EnableRemoteAccess:               true,
		EnableLiveTvManagement:           false,
		EnableLiveTvAccess:               false,
		EnableMediaPlayback:              true,
		EnableAudioPlaybackTranscoding:   true,
		EnableVideoPlaybackTranscoding:   true,
		EnablePlaybackRemuxing:           true,
		ForceRemoteSourceTranscoding:     false,
		EnableContentDeletion:            false,
		EnableContentDeletionFromFolders: []string{},
		EnableContentDownloading:         true,
		EnableSyncTranscoding:            true,
		EnableMediaConversion:            true,
		EnabledDevices:                   []string{},
		EnableAllDevices:                 true,
		EnabledChannels:                  []string{},
		EnableAllChannels:                false,
		EnabledFolders:                   []string{},
		EnableAllFolders:                 true,
		InvalidLoginAttemptCount:         0,
		LoginAttemptsBeforeLockout:       -1,
		MaxActiveSessions:                0,
		EnablePublicSharing:              true,
		BlockedMediaFolders:              []string{},
		BlockedChannels:                  []string{},
		RemoteClientBitrateLimit:         0,
		AuthenticationProviderId:         "",
		PasswordResetProviderId:          "",
		SyncPlayAccess:                   "CreateAndJoinGroups",
	}
}

func userConfiguration() *dto.UserConfiguration {
	return &dto.UserConfiguration{
		PlayDefaultAudioTrack:      true,
		SubtitleLanguagePreference: "",
		DisplayMissingEpisodes:     false,
		GroupedFolders:             []string{},
		SubtitleMode:               "Default",
		DisplayCollectionsView:     false,
		EnableLocalPassword:        false,
		OrderedViews:               []string{},
		LatestItemsExcludes:        []string{},
		MyMediaExcludes:            []string{},
		HidePlayedInLatest:         true,
		RememberAudioSelections:    true,
		RememberSubtitleSelections: true,
		EnableNextEpisodeAutoPlay:  true,
		CastReceiverId:             "",
	}
}
