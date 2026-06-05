package persistence

import (
	"github.com/pocketbase/dbx"
)

// cleanupPluginUserReferences removes a user ID from all plugins' users JSON arrays
// and auto-disables plugins that lose their only permitted user (when users permission is required).
// This is called from userRepository.Delete() to maintain referential integrity.
func cleanupPluginUserReferences(db dbx.Builder, userID string) error {
	// SQLite JSON function: json_remove removes the element at the path where user matches.
	// We use a subquery with json_each to find and remove the user ID from the array.
	// This updates all plugins where the users array contains the given user ID.
	_, err := db.NewQuery(`
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
	`).Bind(dbx.Params{"userID": userID}).Execute()
	if err != nil {
		return err
	}

	// Auto-disable plugins that:
	// 1. Are currently enabled
	// 2. Require users permission (manifest has permissions.users)
	// 3. Don't have allUsers enabled
	// 4. Now have an empty users array after cleanup
	//
	// The manifest check uses JSON path to see if permissions.users exists.
	_, err = db.NewQuery(`
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_users = false
		  AND json_extract(manifest, '$.permissions.users') IS NOT NULL
		  AND (users IS NULL OR users = '' OR users = '[]' OR json_array_length(users) = 0)
	`).Execute()
	return err
}

// cleanupPluginLibraryReferences removes a library ID from all plugins' libraries JSON arrays
// and auto-disables plugins that lose their only permitted library (when library permission is required).
// This is called from libraryRepository.Delete() to maintain referential integrity.
func cleanupPluginLibraryReferences(db dbx.Builder, libraryID int) error {
	// SQLite JSON function: we filter out the library ID from the array.
	// Libraries are stored as integers in the JSON array.
	_, err := db.NewQuery(`
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
	`).Bind(dbx.Params{"libraryID": libraryID}).Execute()
	if err != nil {
		return err
	}

	// Auto-disable plugins that:
	// 1. Are currently enabled
	// 2. Require library permission (manifest has permissions.library)
	// 3. Don't have allLibraries enabled
	// 4. Now have an empty libraries array after cleanup
	_, err = db.NewQuery(`
		UPDATE plugin
		SET enabled = false,
		    updated_at = CURRENT_TIMESTAMP
		WHERE enabled = true
		  AND all_libraries = false
		  AND json_extract(manifest, '$.permissions.library') IS NOT NULL
		  AND (libraries IS NULL OR libraries = '' OR libraries = '[]' OR json_array_length(libraries) = 0)
	`).Execute()
	return err
}
