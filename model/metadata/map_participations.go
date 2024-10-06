package metadata

import (
	"cmp"
	"sync"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

type roleTags struct {
	sort model.TagName
	mbid model.TagName
	name model.TagName
}

func (md Metadata) mapParticipations() model.Participations {
	roleMappings := sync.OnceValue(func() map[model.Role]roleTags {
		return map[model.Role]roleTags{
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
	})

	participations := make(model.Participations)

	// Parse track artists
	artists := md.parseArtists(model.TagTrackArtist, model.TagTrackArtists, model.TagTrackArtistSort, model.TagTrackArtistsSort, model.TagMusicBrainzArtistID)
	participations.Add(model.RoleArtist, artists...)

	// Parse album artists
	albumArtists := md.parseArtists(model.TagAlbumArtist, model.TagAlbumArtists, model.TagAlbumArtistSort, model.TagAlbumArtistsSort, model.TagMusicBrainzAlbumArtistID)
	if len(albumArtists) == 1 && albumArtists[0].Name == consts.UnknownArtist {
		if md.Bool(model.TagCompilation) {
			albumArtists = md.parseArtist([]string{consts.VariousArtists}, nil, []string{consts.VariousArtistsMbzId})
		} else {
			albumArtists = artists
		}
	}
	participations.Add(model.RoleAlbumArtist, albumArtists...)

	// Parse all other roles
	for role, info := range roleMappings() {
		names := md.getTags(info.name)
		sorts := md.getTags(info.sort)
		mbids := md.Strings(info.mbid)
		artists := md.parseArtist(names, sorts, mbids)
		participations.Add(role, artists...)
	}

	// For each artist in each role, try to figure out their MBID from the track/album artists
	for role, participants := range participations {
		for i, participant := range participants {
			if participant.MbzArtistID == "" {
				for _, artist := range append(participations[model.RoleArtist], participations[model.RoleAlbumArtist]...) {
					if participant.Name == artist.Name && artist.MbzArtistID != "" {
						participations[role][i].MbzArtistID = artist.MbzArtistID
						break
					}
				}
			}
		}
	}

	return participations
}

func (md Metadata) parseArtists(name model.TagName, names model.TagName, sort model.TagName, sorts model.TagName, mbid model.TagName) []model.Artist {
	nameValues := md.getTags(names, name)
	sortValues := md.getTags(sorts, sort)
	mbids := md.Strings(mbid)
	if len(nameValues) == 0 {
		nameValues = []string{consts.UnknownArtist}
	}
	return md.parseArtist(nameValues, sortValues, mbids)
}

func (md Metadata) parseArtist(names, sorts, mbids []string) []model.Artist {
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
