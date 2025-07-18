package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

const bookmarkTable = "bookmark"

func (r sqlRepository) withBookmark(query SelectBuilder, idField string) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return query
	}
	return query.
		LeftJoin("bookmark on (" +
			"bookmark.item_id = " + idField +
			" AND bookmark.user_id = '" + userID + "')").
		Columns("coalesce(position, 0) as bookmark_position")
}

func (r sqlRepository) bmkID(itemID ...string) And {
	return And{
		Eq{bookmarkTable + ".user_id": loggedUser(r.ctx).ID},
		Eq{bookmarkTable + ".item_type": r.tableName},
		Eq{bookmarkTable + ".item_id": itemID},
	}
}

func (r sqlRepository) bmkUpsert(itemID, comment string, position int64) error {
	client, _ := request.ClientFrom(r.ctx)
	user, _ := request.UserFrom(r.ctx)
	values := map[string]interface{}{
		"comment":    comment,
		"position":   position,
		"updated_at": time.Now(),
		"changed_by": client,
	}

	upd := Update(bookmarkTable).Where(r.bmkID(itemID)).SetMap(values)
	c, err := r.executeSQL(upd)
	if err == nil {
		log.Debug(r.ctx, "Updated bookmark", "id", itemID, "user", user.UserName, "position", position, "comment", comment)
	}
	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		values["user_id"] = user.ID
		values["item_type"] = r.tableName
		values["item_id"] = itemID
		values["created_at"] = time.Now()
		values["updated_at"] = time.Now()
		ins := Insert(bookmarkTable).SetMap(values)
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
		log.Debug(r.ctx, "Added bookmark", "id", itemID, "user", user.UserName, "position", position, "comment", comment)
	}

	return err
}

func (r sqlRepository) AddBookmark(id, comment string, position int64) error {
	user, _ := request.UserFrom(r.ctx)
	err := r.bmkUpsert(id, comment, position)
	if err != nil {
		log.Error(r.ctx, "Error adding bookmark", "id", id, "user", user.UserName, "position", position, "comment", comment)
	}
	return err
}

func (r sqlRepository) DeleteBookmark(id string) error {
	user, _ := request.UserFrom(r.ctx)
	del := Delete(bookmarkTable).Where(r.bmkID(id))
	_, err := r.executeSQL(del)
	if err != nil {
		log.Error(r.ctx, "Error removing bookmark", "id", id, "user", user.UserName)
	}
	return err
}

type bookmark struct {
	UserID    string    `json:"user_id"`
	ItemID    string    `json:"item_id"`
	ItemType  string    `json:"item_type"`
	Comment   string    `json:"comment"`
	Position  int64     `json:"position"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (r sqlRepository) GetBookmarks() (model.Bookmarks, error) {
	user, _ := request.UserFrom(r.ctx)

	idField := r.tableName + ".id"
	sq := r.newSelect().Columns(r.tableName + ".*")
	sq = r.withAnnotation(sq, idField)
	sq = r.withBookmark(sq, idField).Where(NotEq{bookmarkTable + ".item_id": nil})
	var mfs dbMediaFiles // TODO Decouple from media_file
	err := r.queryAll(sq, &mfs)
	if err != nil {
		log.Error(r.ctx, "Error getting mediafiles with bookmarks", "user", user.UserName, err)
		return nil, err
	}

	ids := make([]string, len(mfs))
	mfMap := make(map[string]int)
	for i, mf := range mfs {
		ids[i] = mf.ID
		mfMap[mf.ID] = i
	}

	sq = Select("*").From(bookmarkTable).Where(r.bmkID(ids...))
	var bmks []bookmark
	err = r.queryAll(sq, &bmks)
	if err != nil {
		log.Error(r.ctx, "Error getting bookmarks", "user", user.UserName, "ids", ids, err)
		return nil, err
	}

	resp := make(model.Bookmarks, len(bmks))
	for i, bmk := range bmks {
		if itemIdx, ok := mfMap[bmk.ItemID]; !ok {
			log.Debug(r.ctx, "Invalid bookmark", "id", bmk.ItemID, "user", user.UserName)
			continue
		} else {
			resp[i] = model.Bookmark{
				Comment:   bmk.Comment,
				Position:  bmk.Position,
				CreatedAt: bmk.CreatedAt,
				UpdatedAt: bmk.UpdatedAt,
				ChangedBy: bmk.ChangedBy,
				Item:      *mfs[itemIdx].MediaFile,
			}
		}
	}
	return resp, nil
}

func (r sqlRepository) cleanBookmarks() error {
	del := Delete(bookmarkTable).Where(Eq{"item_type": r.tableName}).Where("item_id not in (select id from " + r.tableName + ")")
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("error cleaning up bookmarks: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up bookmarks", "totalDeleted", c)
	}
	return nil
}
