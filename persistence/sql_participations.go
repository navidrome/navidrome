package persistence

import (
	"encoding/json"
	"fmt"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type participant struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	SubRole string `json:"subRole,omitempty"`
}

// flatParticipant represents a flattened participant structure for SQL processing
type flatParticipant struct {
	ArtistID string `json:"artist_id"`
	Role     string `json:"role"`
	SubRole  string `json:"sub_role,omitempty"`
}

func marshalParticipants(participants model.Participants) string {
	dbParticipants := make(map[model.Role][]participant)
	for role, artists := range participants {
		for _, artist := range artists {
			dbParticipants[role] = append(dbParticipants[role], participant{ID: artist.ID, SubRole: artist.SubRole, Name: artist.Name})
		}
	}
	res, _ := json.Marshal(dbParticipants)
	return string(res)
}

func unmarshalParticipants(data string) (model.Participants, error) {
	var dbParticipants map[model.Role][]participant
	err := json.Unmarshal([]byte(data), &dbParticipants)
	if err != nil {
		return nil, fmt.Errorf("parsing participants: %w", err)
	}

	participants := make(model.Participants, len(dbParticipants))
	for role, participantList := range dbParticipants {
		artists := slice.Map(participantList, func(p participant) model.Participant {
			return model.Participant{Artist: model.Artist{ID: p.ID, Name: p.Name}, SubRole: p.SubRole}
		})
		participants[role] = artists
	}
	return participants, nil
}

func (r sqlRepository) updateParticipants(itemID string, participants model.Participants) error {
	ids := participants.AllIDs()
	sqd := Delete(r.tableName + "_artists").Where(And{Eq{r.tableName + "_id": itemID}, NotEq{"artist_id": ids}})
	_, err := r.executeSQL(sqd)
	if err != nil {
		return err
	}
	if len(participants) == 0 {
		return nil
	}

	var flatParticipants []flatParticipant
	for role, artists := range participants {
		for _, artist := range artists {
			flatParticipants = append(flatParticipants, flatParticipant{
				ArtistID: artist.ID,
				Role:     role.String(),
				SubRole:  artist.SubRole,
			})
		}
	}

	participantsJSON, err := json.Marshal(flatParticipants)
	if err != nil {
		return fmt.Errorf("marshaling participants: %w", err)
	}

	// Build the INSERT query using json_each and INNER JOIN to artist table
	// to automatically filter out non-existent artist IDs
	query := fmt.Sprintf(`
		INSERT INTO %[1]s_artists (%[1]s_id, artist_id, role, sub_role)
		SELECT ?, 
		       json_extract(value, '$.artist_id') as artist_id,
		       json_extract(value, '$.role') as role,
		       COALESCE(json_extract(value, '$.sub_role'), '') as sub_role
		-- Parse the flat JSON array: [{"artist_id": "id", "role": "role", "sub_role": "subRole"}]
		FROM json_each(?)                                        -- Iterate through each array element
		-- CRITICAL: Only insert records for artists that actually exist in the database
		JOIN artist ON artist.id = json_extract(value, '$.artist_id')  -- Filter out non-existent artist IDs via INNER JOIN
		-- Handle duplicate insertions gracefully (e.g., if called multiple times)
		ON CONFLICT (artist_id, %[1]s_id, role, sub_role) DO NOTHING   -- Ignore duplicates
	`, r.tableName)

	_, err = r.executeSQL(Expr(query, itemID, string(participantsJSON)))
	return err
}

func (r *sqlRepository) getParticipants(m *model.MediaFile) (model.Participants, error) {
	ar := NewArtistRepository(r.ctx, r.db)
	ids := m.Participants.AllIDs()
	artists, err := ar.GetAll(model.QueryOptions{Filters: Eq{"artist.id": ids}})
	if err != nil {
		return nil, fmt.Errorf("getting participants: %w", err)
	}
	artistMap := slice.ToMap(artists, func(a model.Artist) (string, model.Artist) {
		return a.ID, a
	})
	p := m.Participants
	for role, artistList := range p {
		for idx, artist := range artistList {
			if a, ok := artistMap[artist.ID]; ok {
				p[role][idx].Artist = a
			}
		}
	}
	return p, nil
}
