package model

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/utils/slice"
)

var (
	RoleInvalid     = Role{"invalid"}
	RoleArtist      = Role{"artist"}
	RoleAlbumArtist = Role{"albumartist"}
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
	// RoleMainCredit is a credit where the artist is an album artist or artist
	RoleMainCredit = Role{"maincredit"}
)

var AllRoles = map[string]Role{
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
	RoleMainCredit.role:  RoleMainCredit,
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
	if r, ok := AllRoles[role]; ok {
		return r
	}
	return RoleInvalid
}

type Participant struct {
	Artist
	SubRole string `json:"subRole,omitempty"`
}

type ParticipantList []Participant

func (p ParticipantList) Join(sep string) string {
	return strings.Join(slice.Map(p, func(p Participant) string {
		if p.SubRole != "" {
			return p.Name + " (" + p.SubRole + ")"
		}
		return p.Name
	}), sep)
}

type Participants map[Role]ParticipantList

// Add adds the artists to the role, ignoring duplicates.
func (p Participants) Add(role Role, artists ...Artist) {
	participants := slice.Map(artists, func(artist Artist) Participant {
		return Participant{Artist: artist}
	})
	p.add(role, participants...)
}

// AddWithSubRole adds the artists to the role, ignoring duplicates.
func (p Participants) AddWithSubRole(role Role, subRole string, artists ...Artist) {
	participants := slice.Map(artists, func(artist Artist) Participant {
		return Participant{Artist: artist, SubRole: subRole}
	})
	p.add(role, participants...)
}

func (p Participants) Sort() {
	for _, artists := range p {
		slices.SortFunc(artists, func(a1, a2 Participant) int {
			return cmp.Compare(a1.Name, a2.Name)
		})
	}
}

// First returns the first artist for the role, or an empty artist if the role is not present.
func (p Participants) First(role Role) Artist {
	if artists, ok := p[role]; ok && len(artists) > 0 {
		return artists[0].Artist
	}
	return Artist{}
}

// Merge merges the other Participants into this one.
func (p Participants) Merge(other Participants) {
	for role, artists := range other {
		p.add(role, artists...)
	}
}

func (p Participants) add(role Role, participants ...Participant) {
	seen := make(map[string]struct{}, len(p[role]))
	for _, artist := range p[role] {
		seen[artist.ID+artist.SubRole] = struct{}{}
	}
	for _, participant := range participants {
		key := participant.ID + participant.SubRole
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			p[role] = append(p[role], participant)
		}
	}
}

// AllArtists returns all artists found in the Participants.
func (p Participants) AllArtists() []Artist {
	// First count the total number of artists to avoid reallocations.
	totalArtists := 0
	for _, roleArtists := range p {
		totalArtists += len(roleArtists)
	}
	artists := make(Artists, 0, totalArtists)
	for _, roleArtists := range p {
		artists = append(artists, slice.Map(roleArtists, func(p Participant) Artist { return p.Artist })...)
	}
	slices.SortStableFunc(artists, func(a1, a2 Artist) int {
		return cmp.Compare(a1.ID, a2.ID)
	})
	return slices.CompactFunc(artists, func(a1, a2 Artist) bool {
		return a1.ID == a2.ID
	})
}

// AllIDs returns all artist IDs found in the Participants.
func (p Participants) AllIDs() []string {
	artists := p.AllArtists()
	return slice.Map(artists, func(a Artist) string { return a.ID })
}

// AllNames returns all artist names found in the Participants, including SortArtistNames.
func (p Participants) AllNames() []string {
	names := make([]string, 0, len(p))
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

func (p Participants) Hash() []byte {
	flattened := make([]string, 0, len(p))
	for role, artists := range p {
		ids := slice.Map(artists, func(participant Participant) string { return participant.SubRole + ":" + participant.ID })
		slices.Sort(ids)
		flattened = append(flattened, role.String()+":"+strings.Join(ids, "/"))
	}
	slices.Sort(flattened)
	sum := md5.New()
	sum.Write([]byte(strings.Join(flattened, "|")))
	return sum.Sum(nil)
}
