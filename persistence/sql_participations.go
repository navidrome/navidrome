package persistence

import (
	"encoding/json"
	"fmt"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type modelWithParticipations interface {
	getParticipantIDs() []string
	setParticipations(participantsMap map[string]model.Artist)
}

func (r sqlRepository) loadParticipations(m modelWithParticipations) error {
	participantIds := m.getParticipantIDs()
	if len(participantIds) == 0 {
		return nil
	}
	query := Select("id", "name", "mbz_artist_id").From("artist").Where(Eq{"id": participantIds})
	var res model.Artists
	err := r.queryAll(query, &res)
	if err != nil {
		return err
	}
	participantMap := slice.ToMap(res, func(a model.Artist) (string, model.Artist) {
		return a.ID, a
	})
	m.setParticipations(participantMap)
	return nil
}

type participant struct {
	ID      string `json:"id"`
	SubRole string `json:"subRole,omitempty"`
}

func marshalParticipantIDs(participations model.Participations) string {
	ids := make(map[model.Role][]participant)
	for role, artists := range participations {
		for _, artist := range artists {
			ids[role] = append(ids[role], participant{ID: artist.ID, SubRole: artist.SubRole})
		}
	}
	res, _ := json.Marshal(ids)
	return string(res)
}

func unmarshalParticipations(participantIds string) (model.Participations, error) {
	partIDs := make(map[model.Role][]participant)
	err := json.Unmarshal([]byte(participantIds), &partIDs)
	if err != nil {
		return nil, fmt.Errorf("parsing participants: %w", err)
	}

	participations := make(model.Participations, len(partIDs))
	for role, participants := range partIDs {
		artists := slice.Map(participants, func(p participant) model.Participant {
			return model.Participant{Artist: model.Artist{ID: p.ID}, SubRole: p.SubRole}
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
