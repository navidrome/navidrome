package metadata

import (
	"cmp"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils/str"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type roleTags struct {
	name   model.TagName
	sort   model.TagName
	mbid   model.TagName
	credit model.TagName
}

var roleMappings = map[model.Role]roleTags{
	model.RoleComposer:  {name: model.TagComposer, sort: model.TagComposerSort, mbid: model.TagMusicBrainzComposerID, credit: model.TagComposerCredit},
	model.RoleLyricist:  {name: model.TagLyricist, sort: model.TagLyricistSort, mbid: model.TagMusicBrainzLyricistID, credit: model.TagLyricistCredit},
	model.RoleConductor: {name: model.TagConductor, mbid: model.TagMusicBrainzConductorID, credit: model.TagConductorCredit},
	model.RoleArranger:  {name: model.TagArranger, mbid: model.TagMusicBrainzArrangerID, credit: model.TagArrangerCredit},
	model.RoleDirector:  {name: model.TagDirector, mbid: model.TagMusicBrainzDirectorID, credit: model.TagDirectorCredit},
	model.RoleProducer:  {name: model.TagProducer, mbid: model.TagMusicBrainzProducerID, credit: model.TagProducerCredit},
	model.RoleEngineer:  {name: model.TagEngineer, mbid: model.TagMusicBrainzEngineerID, credit: model.TagEngineerCredit},
	model.RoleMixer:     {name: model.TagMixer, mbid: model.TagMusicBrainzMixerID, credit: model.TagMixerCredit},
	model.RoleRemixer:   {name: model.TagRemixer, mbid: model.TagMusicBrainzRemixerID, credit: model.TagRemixerCredit},
	model.RoleDJMixer:   {name: model.TagDJMixer, mbid: model.TagMusicBrainzDJMixerID, credit: model.TagDJMixerCredit},
}

func (md Metadata) mapParticipants() model.Participants {
	participants := make(model.Participants)

	// Parse track artists. MBIDs use getRoleValues so they're split by the same
	// separators ('/' or ';') as the canonical names — otherwise a tag like
	// MUSICBRAINZ_ARTISTID="abc/def" paired with ARTISTS="A/B" would yield 2
	// names but 1 MBID and the positional alignment would break.
	trackNames := md.getArtistValues(model.TagTrackArtist, model.TagTrackArtists)
	if len(trackNames) == 0 {
		trackNames = []string{consts.UnknownArtist}
	}
	trackSorts := md.getArtistValues(model.TagTrackArtistSort, model.TagTrackArtistsSort)
	trackMbids := md.getRoleValues(model.TagMusicBrainzArtistID)
	trackCredits := md.getArtistValues(model.TagTrackArtistCredit, model.TagTrackArtistsCredit)
	trackArtistParticipants := md.buildParticipants(trackNames, trackSorts, trackMbids, trackCredits)
	participants.AddParticipants(model.RoleArtist, trackArtistParticipants...)

	// Parse album artists
	albumNames := md.getArtistValues(model.TagAlbumArtist, model.TagAlbumArtists)
	albumSorts := md.getArtistValues(model.TagAlbumArtistSort, model.TagAlbumArtistsSort)
	albumMbids := md.getRoleValues(model.TagMusicBrainzAlbumArtistID)
	albumCredits := md.getArtistValues(model.TagAlbumArtistCredit, model.TagAlbumArtistsCredit)

	// Treat both "no albumartist tag" and "albumartist tag literally set to
	// the UnknownArtist placeholder" as missing. Preserves behavioral parity
	// with the prior parseArtists path (replaced in this branch), which
	// substituted UnknownArtist on its own and then matched either case
	// downstream. No concrete tagger known to emit the literal '[Unknown
	// Artist]' string, but the cost of keeping the second clause is one
	// comparison and it defends against the placeholder round-tripping if
	// Navidrome's own UnknownArtist value ever ends up back in a tag.
	albumArtistMissing := len(albumNames) == 0 ||
		(len(albumNames) == 1 && albumNames[0] == consts.UnknownArtist)

	var albumArtistParticipants []model.Participant
	if albumArtistMissing {
		if md.Bool(model.TagCompilation) {
			albumArtistParticipants = md.buildParticipants(
				[]string{consts.VariousArtists}, nil,
				[]string{consts.VariousArtistsMbzId}, nil)
		} else {
			albumArtistParticipants = trackArtistParticipants
		}
	} else {
		albumArtistParticipants = md.buildParticipants(albumNames, albumSorts, albumMbids, albumCredits)
	}
	participants.AddParticipants(model.RoleAlbumArtist, albumArtistParticipants...)

	// Parse all other roles. All parallel lists go through getRoleValues so
	// they're split with the same separators as the canonical names. Reading
	// any of these with md.Strings (no splitting) would desync the positional
	// alignment for tags like COMPOSER="A;B" + COMPOSER_CREDIT="AA;BB".
	for role, info := range roleMappings {
		names := md.getRoleValues(info.name)
		if len(names) > 0 {
			sorts := md.getRoleValues(info.sort)
			mbids := md.getRoleValues(info.mbid)
			credits := md.getRoleValues(info.credit)
			participants.AddParticipants(role, md.buildParticipants(names, sorts, mbids, credits)...)
		}
	}

	rolesMbzIdMap := md.buildRoleMbidMaps()
	md.processPerformers(participants, rolesMbzIdMap)
	md.syncMissingMbzIDs(participants)

	return participants
}

// buildRoleMbidMaps creates a map of roles to MBZ IDs
func (md Metadata) buildRoleMbidMaps() map[string][]string {
	titleCaser := cases.Title(language.Und)
	rolesMbzIdMap := make(map[string][]string)
	for _, mbid := range md.Pairs(model.TagMusicBrainzPerformerID) {
		role := titleCaser.String(mbid.Key())
		rolesMbzIdMap[role] = append(rolesMbzIdMap[role], mbid.Value())
	}

	return rolesMbzIdMap
}

func (md Metadata) processPerformers(participants model.Participants, rolesMbzIdMap map[string][]string) {
	// roleIdx keeps track of the index of the MBZ ID for each role
	roleIdx := make(map[string]int)
	for role := range rolesMbzIdMap {
		roleIdx[role] = 0
	}

	titleCaser := cases.Title(language.Und)
	for _, performer := range md.Pairs(model.TagPerformer) {
		name := performer.Value()
		subRole := titleCaser.String(performer.Key())

		artist := model.Artist{
			Name:            name,
			OrderArtistName: str.SanitizeFieldForSortingNoArticle(name),
			MbzArtistID:     md.getPerformerMbid(subRole, rolesMbzIdMap, roleIdx),
		}
		artist.ID = computeArtistPID(
			model.Participant{Artist: artist},
			conf.Server.PID.Artist,
			id.NewHash,
		)
		participants.AddParticipants(model.RolePerformer, model.Participant{
			Artist:     artist,
			SubRole:    subRole,
			CreditedAs: name,
		})
	}
}

// getPerformerMbid returns the MBZ ID for a performer, based on the subrole
func (md Metadata) getPerformerMbid(subRole string, rolesMbzIdMap map[string][]string, roleIdx map[string]int) string {
	if mbids, exists := rolesMbzIdMap[subRole]; exists && roleIdx[subRole] < len(mbids) {
		defer func() { roleIdx[subRole]++ }()
		return mbids[roleIdx[subRole]]
	}
	return ""
}

// syncMissingMbzIDs fills in missing MBZ IDs for artists that have been previously parsed
func (md Metadata) syncMissingMbzIDs(participants model.Participants) {
	artistMbzIDMap := make(map[string]string)
	for _, artist := range append(participants[model.RoleArtist], participants[model.RoleAlbumArtist]...) {
		if artist.MbzArtistID != "" {
			artistMbzIDMap[artist.Name] = artist.MbzArtistID
		}
	}

	for role, list := range participants {
		for i, artist := range list {
			if artist.MbzArtistID == "" {
				if mbzID, exists := artistMbzIDMap[artist.Name]; exists {
					participants[role][i].MbzArtistID = mbzID
				}
			}
		}
	}
}

// buildParticipants builds Artists and wraps each into a Participant with
// CreditedAs populated. credits is paired positionally; if the lengths don't
// match, CreditedAs falls back to the canonical name for every entry.
func (md Metadata) buildParticipants(names, sorts, mbids, credits []string) []model.Participant {
	if len(credits) != 0 && len(credits) != len(names) {
		credits = nil
	}
	artists := md.buildArtists(names, sorts, mbids)
	out := make([]model.Participant, len(artists))
	for i, a := range artists {
		p := model.Participant{Artist: a}
		if i < len(credits) && credits[i] != "" {
			p.CreditedAs = credits[i]
		} else {
			p.CreditedAs = a.Name
		}
		out[i] = p
	}
	return out
}

func (md Metadata) buildArtists(names, sorts, mbids []string) []model.Artist {
	var artists []model.Artist
	for i, name := range names {
		artist := model.Artist{
			Name:            name,
			OrderArtistName: str.SanitizeFieldForSortingNoArticle(name),
		}
		if i < len(sorts) {
			artist.SortArtistName = sorts[i]
		}
		if i < len(mbids) {
			artist.MbzArtistID = mbids[i]
		}
		// Compute ID from the participant fields we just populated so MBID/sort-based specs work.
		artist.ID = computeArtistPID(
			model.Participant{Artist: artist},
			conf.Server.PID.Artist,
			id.NewHash,
		)
		artists = append(artists, artist)
	}
	return artists
}

// getRoleValues returns the values of a role tag, splitting them if necessary
func (md Metadata) getRoleValues(role model.TagName) []string {
	values := md.Strings(role)
	if len(values) == 0 {
		return nil
	}
	conf := model.TagMainMappings()[role]
	if conf.Split == nil {
		conf = model.TagRolesConf()
	}
	if len(conf.Split) > 0 {
		values = conf.SplitTagValue(values)
		return filterDuplicatedOrEmptyValues(values)
	}
	return values
}

// getArtistValues returns the values of a single or multi artist tag, splitting them if necessary
func (md Metadata) getArtistValues(single, multi model.TagName) []string {
	vMulti := md.Strings(multi)
	if len(vMulti) > 0 {
		return vMulti
	}
	vSingle := md.Strings(single)
	if len(vSingle) != 1 {
		return vSingle
	}
	conf := model.TagMainMappings()[single]
	if conf.Split == nil {
		conf = model.TagArtistsConf()
	}
	if len(conf.Split) > 0 {
		vSingle = conf.SplitTagValue(vSingle)
		return filterDuplicatedOrEmptyValues(vSingle)
	}
	return vSingle
}

func (md Metadata) mapDisplayName(singularTagName, pluralTagName model.TagName) string {
	return cmp.Or(
		strings.Join(md.tags[singularTagName], conf.Server.Scanner.ArtistJoiner),
		strings.Join(md.tags[pluralTagName], conf.Server.Scanner.ArtistJoiner),
	)
}

func (md Metadata) mapDisplayArtist() string {
	return cmp.Or(
		md.mapDisplayName(model.TagTrackArtist, model.TagTrackArtists),
		consts.UnknownArtist,
	)
}

func (md Metadata) mapDisplayAlbumArtist(mf model.MediaFile) string {
	fallbackName := consts.UnknownArtist
	if md.Bool(model.TagCompilation) {
		fallbackName = consts.VariousArtists
	}
	return cmp.Or(
		md.mapDisplayName(model.TagAlbumArtist, model.TagAlbumArtists),
		mf.Participants.First(model.RoleAlbumArtist).Name,
		fallbackName,
	)
}
