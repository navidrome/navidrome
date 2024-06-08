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

func (md Metadata) participations() model.Participations {
	roleMappings := sync.OnceValue(func() map[model.Role]artistInfo {
		return map[model.Role]artistInfo{
			//model.RoleArtist:      {name: TrackArtist, sort: TrackArtistSort, mbid: MusicBrainzArtistID, mapName: md.mapTrackArtistName},
			//model.RoleAlbumArtist: {name: AlbumArtist, sort: AlbumArtistSort, mbid: MusicBrainzAlbumArtistID, mapName: md.mapAlbumArtistName},
			model.RoleComposer:  {name: Composer, sort: ComposerSort},
			model.RoleConductor: {name: Conductor},
			model.RoleLyricist:  {name: Lyricist},
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

	artists := md.parseArtists(TrackArtist, TrackArtists, TrackArtistSort, TrackArtistsSort, MusicBrainzArtistID)
	for _, a := range artists {
		participations.Add(a, model.RoleArtist)
	}

	for role, info := range roleMappings() {
		names := md.getTags(info.name)
		sorts := md.getTags(info.sort)
		mbids := md.Strings(info.mbid)
		artists := md.parseArtist(names, sorts, mbids)
		for _, a := range artists {
			participations.Add(a, role)
		}
	}
	// TODO If track artist is not set, use Unknown Artist (maybe try sort name first?)
	// TODO If album artist is not set, use track artist (maybe try sort name first?)
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

func (md Metadata) mapTrackArtistName(name string) string {
	return cmp.Or(
		name,
		consts.UnknownArtist,
	)
}

func (md Metadata) mapAlbumArtistName(name string) string {
	return cmp.Or(
		name,
		func() string {
			if md.Bool(Compilation) {
				return consts.VariousArtists
			}
			return ""
		}(),
		consts.UnknownArtist,
	)
}

func (md Metadata) displayArtist(mf model.MediaFile) string {
	artistNames := md.Strings(TrackArtist)
	artistsNames := md.Strings(TrackArtists)
	values := []string{
		"",
		mf.Participations.First(model.RoleArtist).Name,
		consts.UnknownArtist,
	}
	if len(artistNames) == 1 {
		values[0] = artistNames[0]
	} else if len(artistsNames) > 0 {
		values[0] = artistsNames[0]
	} else if len(artistNames) > 0 {
		values[0] = artistNames[0]
	}
	return cmp.Or(values...)
}

// TODO incomplete
func (md Metadata) displayAlbumArtist(mf model.MediaFile) string {
	return mf.Artist
}
