package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upEncodeAllPasswords, downEncodeAllPasswords)
}

func upEncodeAllPasswords(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id, user_name, password from user;`)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare("UPDATE user SET password = ? WHERE id = ?")
	if err != nil {
		return err
	}
	var id string
	var username, password string

	data := sha256.Sum256([]byte(consts.DefaultEncryptionKey))
	encKey := data[0:]

	for rows.Next() {
		err = rows.Scan(&id, &username, &password)
		if err != nil {
			return err
		}

		password, err = utils.Encrypt(ctx, encKey, password)
		if err != nil {
			log.Error("Error encrypting user's password", "id", id, "username", username, err)
		}

		_, err = stmt.Exec(password, id)
		if err != nil {
			log.Error("Error saving user's encrypted password", "id", id, "username", username, err)
		}
	}
	return rows.Err()
}

func downEncodeAllPasswords(_ context.Context, tx *sql.Tx) error {
	return nil
}
