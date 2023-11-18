package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils/path_hash"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddFolderDb, downAddFolderDb)
}

func upAddFolderDb(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists directory_entry(
	id varchar(255) not null
		primary key,
	path varchar(255) default '' not null,
	name varchar(255) default '' not null,
	parent_id varchar(255) default null
		references directory_entry (id)
		 	on delete cascade
);

create index if not exists directory_entry_parent
	on directory_entry (parent_id);
`)

	if err != nil {
		return err
	}

	rows, err := tx.Query(
		fmt.Sprintf(`SELECT substr(path, %d), id as path FROM media_file order by path;`, len(conf.Server.MusicFolder)+2))
	if err != nil {
		return err
	}

	root_id := path_hash.PathToMd5Hash(conf.Server.MusicFolder)

	var path string
	var id string

	insertDir, err := tx.Prepare("INSERT INTO directory_entry (id, path, name, parent_id) VALUES (?, ?, ?, ?);")
	if err != nil {
		return err
	}

	insertMediaFile, err := tx.Prepare("INSERT INTO directory_entry (id, parent_id) VALUES (?, ?);")
	if err != nil {
		return err
	}

	folders := map[string]string{
		conf.Server.MusicFolder: root_id,
	}
	_, err = tx.Exec(
		`INSERT INTO directory_entry (id, path, name) VALUES (?, ?, ?);`,
		root_id,
		conf.Server.MusicFolder,
		"Music Library",
	)
	if err != nil {
		return err
	}

	for rows.Next() {
		err = rows.Scan(&path, &id)
		if err != nil {
			return err
		}

		dir := filepath.Dir(path)
		parent, ok := folders[dir]
		if !ok {
			paths := strings.Split(dir, string(os.PathSeparator))

			var base_path = conf.Server.MusicFolder
			parent = root_id
			for _, directory := range paths {
				base_path = filepath.Join(base_path, directory)
				next_id, ok := folders[base_path]

				if !ok {
					next_id = path_hash.PathToMd5Hash(base_path)
					_, err := insertDir.Exec(next_id, base_path, directory, parent)
					if err != nil {
						return err
					}
					folders[base_path] = next_id
				}

				parent = next_id
			}
		}

		_, err = insertMediaFile.Exec(id, parent)
		if err != nil {
			return err
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return err
}

func downAddFolderDb(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
