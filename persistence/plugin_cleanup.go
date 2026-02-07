package persistence

import (
	"github.com/navidrome/navidrome/db"
	"github.com/pocketbase/dbx"
)

// cleanupPluginUserReferences removes a user ID from all plugins' users JSON arrays
// and auto-disables plugins that lose their only permitted user.
func cleanupPluginUserReferences(dbConn dbx.Builder, userID string) error {
	updateSQL := `
		UPDATE plugin
		SET users = (
			SELECT json_group_array(value)
			FROM json_each(plugin.users)
			WHERE value != {:userID}
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE users IS NOT NULL
		  AND users != ''
		  AND EXISTS (SELECT 1 FROM json_each(plugin.users) WHERE value = {:userID})
	`
	if db.IsPostgres() {
		updateSQL = `
		UPDATE plugin
		SET users = (
			SELECT json_agg(elem)::text
			FROM jsonb_array_elements_text(plugin.users::jsonb) as elem
			WHERE elem != {:userID}
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE users IS NOT NULL
		  AND users != ''
		  AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(plugin.users::jsonb) as elem WHERE elem = {:userID})
	`
	}
	_, err := dbConn.NewQuery(updateSQL).Bind(dbx.Params{"userID": userID}).Execute()
	if err != nil {
		return err
	}

	// Auto-disable plugins that now have no permitted users left
	disableSQL := `
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_users = false
		  AND json_extract(manifest, '$.permissions.users') IS NOT NULL
		  AND (users IS NULL OR users = '' OR users = '[]' OR json_array_length(users) = 0)
	`
	if db.IsPostgres() {
		disableSQL = `
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_users = false
		  AND manifest::jsonb->'permissions'->'users' IS NOT NULL
		  AND (users IS NULL OR users = '' OR users = '[]' OR jsonb_array_length(users::jsonb) = 0)
	`
	}
	_, err = dbConn.NewQuery(disableSQL).Execute()
	return err
}

// cleanupPluginLibraryReferences removes a library ID from all plugins' libraries JSON arrays
// and auto-disables plugins that lose their only permitted library.
func cleanupPluginLibraryReferences(dbConn dbx.Builder, libraryID int) error {
	updateSQL := `
		UPDATE plugin
		SET libraries = (
			SELECT json_group_array(value)
			FROM json_each(plugin.libraries)
			WHERE CAST(value AS INTEGER) != {:libraryID}
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE libraries IS NOT NULL
		  AND libraries != ''
		  AND EXISTS (SELECT 1 FROM json_each(plugin.libraries) WHERE CAST(value AS INTEGER) = {:libraryID})
	`
	if db.IsPostgres() {
		updateSQL = `
		UPDATE plugin
		SET libraries = (
			SELECT json_agg(elem)::text
			FROM jsonb_array_elements_text(plugin.libraries::jsonb) as elem
			WHERE elem::int != {:libraryID}
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE libraries IS NOT NULL
		  AND libraries != ''
		  AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(plugin.libraries::jsonb) as elem WHERE elem::int = {:libraryID})
	`
	}
	_, err := dbConn.NewQuery(updateSQL).Bind(dbx.Params{"libraryID": libraryID}).Execute()
	if err != nil {
		return err
	}

	// Auto-disable plugins that now have no permitted libraries left
	disableSQL := `
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_libraries = false
		  AND json_extract(manifest, '$.permissions.library') IS NOT NULL
		  AND (libraries IS NULL OR libraries = '' OR libraries = '[]' OR json_array_length(libraries) = 0)
	`
	if db.IsPostgres() {
		disableSQL = `
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_libraries = false
		  AND manifest::jsonb->'permissions'->'library' IS NOT NULL
		  AND (libraries IS NULL OR libraries = '' OR libraries = '[]' OR jsonb_array_length(libraries::jsonb) = 0)
	`
	}
	_, err = dbConn.NewQuery(disableSQL).Execute()
	return err
}
