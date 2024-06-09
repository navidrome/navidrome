package metadata

import (
	"cmp"
	"sync"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

type artistInfo struct {
	sort    TagName
	mbid    TagName
	name    TagName
	mapName func(string) string
}

func (md Metadata) mapParticipations() model.Participations {
	roleMappings := sync.OnceValue(func() map[model.Role]artistInfo {
		return map[model.Role]artistInfo{
			model.RoleComposer:  {name: Composer, sort: ComposerSort},
			model.RoleLyricist:  {name: Lyricist, sort: LyricistSort},
			model.RoleConductor: {name: Conductor},
			model.RoleArranger:  {name: Arranger},
			model.RoleDirector:  {name: Director},
			model.RoleProducer:  {name: Producer},
			model.RoleEngineer:  {name: Engineer},
			model.RoleMixer:     {name: Mixer},
			model.RoleRemixer:   {name: Remixer},
			model.RoleDJMixer:   {name: DJMixer},
			// TODO Performer (and Instruments)
		}
	})

	participations := make(model.Participations)

	// Parse track artists
	artists := md.parseArtists(TrackArtist, TrackArtists, TrackArtistSort, TrackArtistsSort, MusicBrainzArtistID)
	for _, a := range artists {
		participations.Add(a, model.RoleArtist)
	}

	// Parse album artists
	albumArtists := md.parseArtists(AlbumArtist, AlbumArtists, AlbumArtistSort, AlbumArtistsSort, MusicBrainzAlbumArtistID)
	if len(albumArtists) == 1 && albumArtists[0].Name == consts.UnknownArtist {
		if md.Bool(Compilation) {
			albumArtists = md.parseArtist([]string{consts.VariousArtists}, nil, []string{consts.VariousArtistsMbzId})
		} else {
			albumArtists = artists
		}
	}
	for _, a := range albumArtists {
		participations.Add(a, model.RoleAlbumArtist)
	}

	// Parse all other roles
	for role, info := range roleMappings() {
		names := md.getTags(info.name)
		sorts := md.getTags(info.sort)
		mbids := md.Strings(info.mbid)
		artists := md.parseArtist(names, sorts, mbids)
		for _, a := range artists {
			participations.Add(a, role)
		}
	}
	// TODO Match participants by name and copy MBID if not set
	return participations
}

func (md Metadata) parseArtists(name TagName, names TagName, sort TagName, sorts TagName, mbid TagName) []model.Artist {
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

func (md Metadata) getTags(tagNames ...TagName) []string {
	for _, tagName := range tagNames {
		values := md.Strings(tagName)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}
func (md Metadata) mapDisplayRole(mf model.MediaFile, role model.Role, tagNames ...TagName) string {
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
	return md.mapDisplayRole(mf, model.RoleArtist, TrackArtist, TrackArtists)
}

func (md Metadata) mapDisplayAlbumArtist(mf model.MediaFile) string {
	return md.mapDisplayRole(mf, model.RoleAlbumArtist, AlbumArtist, AlbumArtists)
}
