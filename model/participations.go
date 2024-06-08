package model

type Role struct {
	role string
}

var (
	RoleArtist      = Role{"artist"}
	RoleAlbumArtist = Role{"album_artist"}
	RoleComposer    = Role{"composer"}
	RoleConductor   = Role{"conductor"}
	RoleLyricist    = Role{"lyricist"}
	RoleArranger    = Role{"arranger"}
	RoleProducer    = Role{"producer"}
	RoleDirector    = Role{"director"}
	RoleEngineer    = Role{"engineer"}
	RoleMixer       = Role{"mixer"}
	RoleRemixer     = Role{"remixer"}
	RoleDJMixer     = Role{"djmixer"}
	RolePerformer   = Role{"performer"}
)

type Participations map[Role][]Artist

func (p Participations) Add(artist Artist, role Role) {
	if _, ok := p[role]; !ok {
		p[role] = []Artist{}
	}
	p[role] = append(p[role], artist)
}

func (p Participations) First(role Role) Artist {
	if artists, ok := p[role]; ok && len(artists) > 0 {
		return artists[0]
	}
	return Artist{}
}
