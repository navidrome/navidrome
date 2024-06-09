package persistence

import (
	"encoding/json"
	"fmt"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

func (r sqlRepository) withParticipations(sql SelectBuilder) SelectBuilder {
	return sql.LeftJoin(r.tableName + "_artists on " + r.tableName + "_artists." + r.tableName + "_id = " + r.tableName + ".id").
		LeftJoin(fmt.Sprintf("%[1]s_artists r on r.%[1]s_id = %[1]s.id", r.tableName)).
		LeftJoin("artist a on a.id = r.artist_id").
		Columns("json_group_array(distinct(json_object('id', a.id, 'name', a.name, 'role', r.role))) as participations")
}

func parseParticipations(strParticipations string) model.Participations {
	participations := model.Participations{}
	var dbParticipations []map[string]string
	err := json.Unmarshal([]byte(strParticipations), &dbParticipations)
	if err != nil {
		return nil
	}
	for _, p := range dbParticipations {
		mRole := model.RoleFromString(p["role"])
		participations[mRole] = append(participations[mRole], model.Artist{
			ID:   p["id"],
			Name: p["name"],
		})
	}
	return participations
}

func (r sqlRepository) updateParticipations(itemID string, participations model.Participations) error {
	sqd := Delete(r.tableName + "_artists").Where(Eq{r.tableName + "_id": itemID})
	_, err := r.executeSQL(sqd)
	if err != nil {
		return err
	}
	if len(participations) == 0 {
		return nil
	}
	sqi := Insert(r.tableName+"_artists").
		Columns(r.tableName+"_id", "artist_id", "role"). // TODO Sub-role
		Suffix(fmt.Sprintf("on conflict (artist_id, %s_id, role) do nothing", r.tableName))
	for role, artists := range participations {
		for _, artist := range artists {
			sqi = sqi.Values(itemID, artist.ID, role.String())
		}
	}
	_, err = r.executeSQL(sqi)
	return err
}
