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

	// Use json_tree with INNER JOIN to artist table to automatically filter out non-existent artist IDs
	participantsJSON, err := json.Marshal(participants)
	if err != nil {
		return fmt.Errorf("marshaling participants: %w", err)
	}

	// Build the INSERT query using json_tree and INNER JOIN
	query := fmt.Sprintf(`
		INSERT INTO %s_artists (%s_id, artist_id, role, sub_role)
		SELECT ?, 
		       participant_data.value as artist_id,
		       role_name.key as role,
		       COALESCE(sub_role_data.value, '') as sub_role
		FROM json_tree(?, '$') as role_name
		JOIN json_tree(role_name.value, '$') as participant_idx ON participant_idx.type = 'object'
		JOIN json_tree(participant_idx.value, '$.id') as participant_data ON participant_data.atom IS NOT NULL
		LEFT JOIN json_tree(participant_idx.value, '$.subRole') as sub_role_data ON sub_role_data.atom IS NOT NULL
		JOIN artist ON artist.id = participant_data.value
		WHERE role_name.type = 'array'
		ON CONFLICT (artist_id, %s_id, role, sub_role) DO NOTHING
	`, r.tableName, r.tableName, r.tableName)

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
