package model

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/navidrome/navidrome/utils/slice"
)

var (
	RoleInvalid     = Role{"invalid"}
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

var allRoles = map[string]Role{
	RoleArtist.role:      RoleArtist,
	RoleAlbumArtist.role: RoleAlbumArtist,
	RoleComposer.role:    RoleComposer,
	RoleConductor.role:   RoleConductor,
	RoleLyricist.role:    RoleLyricist,
	RoleArranger.role:    RoleArranger,
	RoleProducer.role:    RoleProducer,
	RoleDirector.role:    RoleDirector,
	RoleEngineer.role:    RoleEngineer,
	RoleMixer.role:       RoleMixer,
	RoleRemixer.role:     RoleRemixer,
	RoleDJMixer.role:     RoleDJMixer,
	RolePerformer.role:   RolePerformer,
}

// Role represents the role of an artist in a track or album.
type Role struct {
	role string
}

func (r Role) String() string {
	return r.role
}

func (r Role) MarshalText() (text []byte, err error) {
	return []byte(r.role), nil
}

func (r *Role) UnmarshalText(text []byte) error {
	role := RoleFromString(string(text))
	if role == RoleInvalid {
		return fmt.Errorf("invalid role: %s", text)
	}
	*r = role
	return nil
}

func RoleFromString(role string) Role {
	if r, ok := allRoles[role]; ok {
		return r
	}
	return RoleInvalid
}

type Participations map[Role][]Artist

// Add adds the artists to the role, ignoring duplicates.
func (p Participations) Add(role Role, artists ...Artist) {
	seen := map[string]struct{}{}
	for _, artist := range p[role] {
		seen[artist.ID] = struct{}{}
	}
	for _, artist := range artists {
		if _, ok := seen[artist.ID]; !ok {
			seen[artist.ID] = struct{}{}
			p[role] = append(p[role], artist)
		}
	}
}

func (p Participations) Sort() {
	for _, artists := range p {
		slices.SortFunc(artists, func(a1, a2 Artist) int {
			return cmp.Compare(a1.Name, a2.Name)
		})
	}
}

// First returns the first artist for the role, or an empty artist if the role is not present.
func (p Participations) First(role Role) Artist {
	if artists, ok := p[role]; ok && len(artists) > 0 {
		return artists[0]
	}
	return Artist{}
}

// Merge merges the other Participations into this one.
func (p Participations) Merge(other Participations) {
	for role, artists := range other {
		p.Add(role, artists...)
	}
}

// AllIDs returns all artist IDs found in the Participations.
func (p Participations) AllIDs() []string {
	var ids []string
	for _, artists := range p {
		for _, artist := range artists {
			ids = append(ids, artist.ID)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// AllNames returns all artist names found in the Participations.
func (p Participations) AllNames() []string {
	var names []string
	for _, artists := range p {
		for _, artist := range artists {
			names = append(names, artist.Name)
			if artist.SortArtistName != "" {
				names = append(names, artist.SortArtistName)
			}
		}
	}
	return slice.Unique(names)
}
