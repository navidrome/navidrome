package model

import "encoding/json"

type Role struct {
	role string
}

func (r Role) String() string {
	return r.role
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

var AllRoles = []Role{
	RoleArtist,
	RoleAlbumArtist,
	RoleComposer,
	RoleConductor,
	RoleLyricist,
	RoleArranger,
	RoleProducer,
	RoleDirector,
	RoleEngineer,
	RoleMixer,
	RoleRemixer,
	RoleDJMixer,
	RolePerformer,
}

func RoleFromString(role string) Role {
	for _, r := range AllRoles {
		if r.String() == role {
			return r
		}
	}
	return Role{}
}

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

func (p *Participations) Merge(other Participations) {
	for role, artists := range other {
		for _, artist := range artists {
			p.Add(artist, role)
		}
	}
}

func (p Participations) MarshalJSON() ([]byte, error) {
	m := map[string][]Artist{}
	for role, artists := range p {
		m[role.String()] = artists
	}
	return json.Marshal(m)
}

func (p *Participations) UnmarshalJSON(data []byte) error {
	m := map[string][]Artist{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for role, artists := range m {
		mRole := RoleFromString(role)
		for _, artist := range artists {
			p.Add(artist, mRole)
		}
	}
	return nil
}
