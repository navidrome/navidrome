package metadata

import (
	"cmp"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type roleTags struct {
	name model.TagName
	sort model.TagName
	mbid model.TagName
}

var roleMappings = map[model.Role]roleTags{
	model.RoleComposer:  {name: model.TagComposer, sort: model.TagComposerSort, mbid: model.TagMusicBrainzComposerID},
	model.RoleLyricist:  {name: model.TagLyricist, sort: model.TagLyricistSort, mbid: model.TagMusicBrainzLyricistID},
	model.RoleConductor: {name: model.TagConductor, mbid: model.TagMusicBrainzConductorID},
	model.RoleArranger:  {name: model.TagArranger, mbid: model.TagMusicBrainzArrangerID},
	model.RoleDirector:  {name: model.TagDirector, mbid: model.TagMusicBrainzDirectorID},
	model.RoleProducer:  {name: model.TagProducer, mbid: model.TagMusicBrainzProducerID},
	model.RoleEngineer:  {name: model.TagEngineer, mbid: model.TagMusicBrainzEngineerID},
	model.RoleMixer:     {name: model.TagMixer, mbid: model.TagMusicBrainzMixerID},
	model.RoleRemixer:   {name: model.TagRemixer, mbid: model.TagMusicBrainzRemixerID},
	model.RoleDJMixer:   {name: model.TagDJMixer, mbid: model.TagMusicBrainzDJMixerID},
}

func (md Metadata) mapParticipants() model.Participants {
	participants := make(model.Participants)

	// Parse track artists
	artists := md.parseArtists(
		model.TagTrackArtist, model.TagTrackArtists,
		model.TagTrackArtistSort, model.TagTrackArtistsSort,
		model.TagMusicBrainzArtistID,
	)
	participants.Add(model.RoleArtist, artists...)

	// Parse album artists
	albumArtists := md.parseArtists(
		model.TagAlbumArtist, model.TagAlbumArtists,
		model.TagAlbumArtistSort, model.TagAlbumArtistsSort,
		model.TagMusicBrainzAlbumArtistID,
	)
	if len(albumArtists) == 1 && albumArtists[0].Name == consts.UnknownArtist {
		if md.Bool(model.TagCompilation) {
			albumArtists = md.buildArtists([]string{consts.VariousArtists}, nil, []string{consts.VariousArtistsMbzId})
		} else {
			albumArtists = artists
		}
	}
	participants.Add(model.RoleAlbumArtist, albumArtists...)

	// Parse all other roles
	for role, info := range roleMappings {
		names := md.getRoleValues(info.name)
		if len(names) > 0 {
			sorts := md.Strings(info.sort)
			mbids := md.Strings(info.mbid)
			artists := md.buildArtists(names, sorts, mbids)
			participants.Add(role, artists...)
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
			ID:              md.artistID(name),
			Name:            name,
			OrderArtistName: str.SanitizeFieldForSortingNoArticle(name),
			MbzArtistID:     md.getPerformerMbid(subRole, rolesMbzIdMap, roleIdx),
		}
		participants.AddWithSubRole(model.RolePerformer, subRole, artist)
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

func (md Metadata) parseArtists(
	name model.TagName, names model.TagName, sort model.TagName,
	sorts model.TagName, mbid model.TagName,
) []model.Artist {
	nameValues := md.getArtistValues(name, names)
	sortValues := md.getArtistValues(sort, sorts)
	mbids := md.Strings(mbid)
	if len(nameValues) == 0 {
		nameValues = []string{consts.UnknownArtist}
	}
	return md.buildArtists(nameValues, sortValues, mbids)
}

func (md Metadata) buildArtists(names, sorts, mbids []string) []model.Artist {
	var artists []model.Artist
	for i, name := range names {
		id := md.artistID(name)
		artist := model.Artist{
			ID:              id,
			Name:            name,
			OrderArtistName: str.SanitizeFieldForSortingNoArticle(name),
		}
		if i < len(sorts) {
			artist.SortArtistName = sorts[i]
		}
		if i < len(mbids) {
			artist.MbzArtistID = mbids[i]
		}
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
