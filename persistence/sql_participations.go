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

func marshalParticipations(participations model.Participations) string {
	dbParticipations := make(map[model.Role][]participant)
	for role, artists := range participations {
		for _, artist := range artists {
			dbParticipations[role] = append(dbParticipations[role], participant{ID: artist.ID, SubRole: artist.SubRole, Name: artist.Name})
		}
	}
	res, _ := json.Marshal(dbParticipations)
	return string(res)
}

func unmarshalParticipations(data string) (model.Participations, error) {
	var dbParticipations map[model.Role][]participant
	err := json.Unmarshal([]byte(data), &dbParticipations)
	if err != nil {
		return nil, fmt.Errorf("parsing participants: %w", err)
	}

	participations := make(model.Participations, len(dbParticipations))
	for role, participants := range dbParticipations {
		artists := slice.Map(participants, func(p participant) model.Participant {
			return model.Participant{Artist: model.Artist{ID: p.ID, Name: p.Name}, SubRole: p.SubRole}
		})
		participations[role] = artists
	}
	return participations, nil
}

func (r sqlRepository) updateParticipations(itemID string, participations model.Participations) error {
	ids := participations.AllIDs()
	sqd := Delete(r.tableName + "_artists").Where(And{Eq{r.tableName + "_id": itemID}, NotEq{"artist_id": ids}})
	_, err := r.executeSQL(sqd)
	if err != nil {
		return err
	}
	if len(participations) == 0 {
		return nil
	}
	sqi := Insert(r.tableName+"_artists").
		Columns(r.tableName+"_id", "artist_id", "role", "sub_role").
		Suffix(fmt.Sprintf("on conflict (artist_id, %s_id, role, sub_role) do nothing", r.tableName))
	for role, artists := range participations {
		for _, artist := range artists {
			sqi = sqi.Values(itemID, artist.ID, role.String(), artist.SubRole)
		}
	}
	_, err = r.executeSQL(sqi)
	return err
}
