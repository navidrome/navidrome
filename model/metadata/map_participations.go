package metadata

import (
	"cmp"

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
	model.RoleComposer:  {name: model.TagComposer, sort: model.TagComposerSort},
	model.RoleLyricist:  {name: model.TagLyricist, sort: model.TagLyricistSort},
	model.RoleConductor: {name: model.TagConductor},
	model.RoleArranger:  {name: model.TagArranger},
	model.RoleDirector:  {name: model.TagDirector},
	model.RoleProducer:  {name: model.TagProducer},
	model.RoleEngineer:  {name: model.TagEngineer},
	model.RoleMixer:     {name: model.TagMixer},
	model.RoleRemixer:   {name: model.TagRemixer},
	model.RoleDJMixer:   {name: model.TagDJMixer},
	// TODO Performer (and Instruments)
}

func (md Metadata) mapParticipations() model.Participations {
	participations := make(model.Participations)

	// Parse track artists
	artists := md.parseArtists(model.TagTrackArtist, model.TagTrackArtists, model.TagTrackArtistSort, model.TagTrackArtistsSort, model.TagMusicBrainzArtistID)
	participations.Add(model.RoleArtist, artists...)

	// Parse album artists
	albumArtists := md.parseArtists(model.TagAlbumArtist, model.TagAlbumArtists, model.TagAlbumArtistSort, model.TagAlbumArtistsSort, model.TagMusicBrainzAlbumArtistID)
	if len(albumArtists) == 1 && albumArtists[0].Name == consts.UnknownArtist {
		if md.Bool(model.TagCompilation) {
			albumArtists = md.buildArtists([]string{consts.VariousArtists}, nil, []string{consts.VariousArtistsMbzId})
		} else {
			albumArtists = artists
		}
	}
	participations.Add(model.RoleAlbumArtist, albumArtists...)

	// Parse all other roles
	for role, info := range roleMappings {
		names := md.getRoleValues(info.name)
		if len(names) > 0 {
			sorts := md.Strings(info.sort)
			mbids := md.Strings(info.mbid)
			artists := md.buildArtists(names, sorts, mbids)
			participations.Add(role, artists...)
		}
	}

	// Parse performers
	caser := cases.Title(language.Und)
	for _, performer := range md.Pairs(model.TagPerformer) {
		name := performer.Value()
		id := md.artistID(name)
		orderName := str.SanitizeFieldForSortingNoArticle(name)
		subRole := caser.String(performer.Key())
		participations.AddWithSubRole(model.RolePerformer, subRole, model.Artist{
			ID:              id,
			Name:            name,
			OrderArtistName: orderName,
		})
	}

	// Create a map to store the MbzArtistID for each artist name
	artistMbzIDMap := make(map[string]string)
	for _, artist := range append(participations[model.RoleArtist], participations[model.RoleAlbumArtist]...) {
		if artist.MbzArtistID != "" {
			artistMbzIDMap[artist.Name] = artist.MbzArtistID
		}
	}

	if len(artistMbzIDMap) > 0 {
		// For each artist in each role, try to figure out their MBID from the
		// track/album artists (the only roles that have MBID in MusicBrainz)
		for role, participants := range participations {
			for i, participant := range participants {
				if participant.MbzArtistID == "" {
					if mbzID, found := artistMbzIDMap[participant.Name]; found {
						participations[role][i].MbzArtistID = mbzID
					}
				}
			}
		}
	}

	return participations
}

func (md Metadata) parseArtists(name model.TagName, names model.TagName, sort model.TagName, sorts model.TagName, mbid model.TagName) []model.Artist {
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
	if conf := rolesConf(); len(conf.Split) > 0 {
		values = split(values, conf.Split)
		return removeDuplicatedAndEmpty(values)
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
	conf := artistsConf()
	if len(conf.Split) > 0 {
		vSingle = split(vSingle, conf.Split)
		return removeDuplicatedAndEmpty(vSingle)
	}
	return vSingle
}

func (md Metadata) getTags(tagNames ...model.TagName) []string {
	for _, tagName := range tagNames {
		values := md.Strings(tagName)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}
func (md Metadata) mapDisplayRole(mf model.MediaFile, role model.Role, tagNames ...model.TagName) string {
	artistNames := md.getTags(tagNames...)
	values := []string{
		"",
		mf.Participations.First(role).Name,
		consts.UnknownArtist,
	}
	if len(artistNames) == 1 {
		values[0] = artistNames[0]
	}
	return cmp.Or(values...)
}

func (md Metadata) mapDisplayArtist(mf model.MediaFile) string {
	return md.mapDisplayRole(mf, model.RoleArtist, model.TagTrackArtist, model.TagTrackArtists)
}

func (md Metadata) mapDisplayAlbumArtist(mf model.MediaFile) string {
	return md.mapDisplayRole(mf, model.RoleAlbumArtist, model.TagAlbumArtist, model.TagAlbumArtists)
}
