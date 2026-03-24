// Package e2e provides end-to-end integration tests for the Navidrome Subsonic API.
//
// These tests exercise the full HTTP request/response cycle through the Subsonic API router,
// using a real SQLite database and real repository implementations while stubbing out external
// services (artwork, streaming, scrobbling, etc.) with noop implementations.
//
// # Test Infrastructure
//
// The suite uses [Ginkgo] v2 as the test runner and [Gomega] for assertions. It is invoked
// through the standard Go test entry point [TestSubsonicE2E], which initializes the test
// environment, creates a temporary SQLite database, and runs the specs.
//
// # Setup and Teardown
//
// During [BeforeSuite], the test infrastructure:
//
//  1. Creates a temporary SQLite database with WAL journal mode.
//  2. Initializes the schema via [db.Init].
//  3. Creates two test users: an admin ("admin") and a regular user ("regular"),
//     both with the password "password".
//  4. Creates a single library ("Music Library") backed by a fake in-memory filesystem
//     (scheme "fake:///music") using the [storagetest] package.
//  5. Populates the filesystem with a set of test tracks spanning multiple artists,
//     albums, genres, and years.
//  6. Runs the scanner to import all metadata into the database.
//  7. Takes a snapshot of the database to serve as a golden baseline for test isolation.
//
// # Test Data
//
// The fake filesystem contains the following music library structure:
//
//	Rock/The Beatles/Abbey Road/
//	  01 - Come Together.mp3  (1969, Rock)
//	  02 - Something.mp3      (1969, Rock)
//	Rock/The Beatles/Help!/
//	  01 - Help.mp3           (1965, Rock)
//	Rock/Led Zeppelin/IV/
//	  01 - Stairway To Heaven.mp3 (1971, Rock)
//	Jazz/Miles Davis/Kind of Blue/
//	  01 - So What.mp3        (1959, Jazz)
//	Pop/
//	  01 - Standalone Track.mp3 (2020, Pop)
//
// # Database Isolation
//
// Before each top-level Describe block, the [setupTestDB] function restores the database
// to its golden snapshot state using SQLite's ATTACH DATABASE mechanism. This copies all
// table data from the snapshot back into the main database, providing each test group with
// a clean, consistent starting state without the overhead of re-scanning the filesystem.
//
// A fresh [subsonic.Router] is also created for each test group, wired with real data store
// repositories and noop stubs for external services:
//
//   - noopArtwork: returns [model.ErrNotFound] for all artwork requests.
//   - noopStreamer: returns [model.ErrNotFound] for all stream requests.
//   - noopArchiver: returns [model.ErrNotFound] for all archive requests.
//   - noopProvider: returns empty results for all external metadata lookups.
//   - noopPlayTracker: silently discards all scrobble events.
//
// # Request Helpers
//
// Tests build HTTP requests using the [buildReq] helper, which constructs a Subsonic API
// request with authentication parameters (username, password, API version "1.16.1", client
// name "test-client", and JSON format). Convenience wrappers include:
//
//   - [doReq]: sends a request as the admin user and returns the parsed JSON response.
//   - [doReqWithUser]: sends a request as a specific user.
//   - [doRawReq] / [doRawReqWithUser]: returns the raw [httptest.ResponseRecorder] for
//     binary content or status code inspection.
//
// Responses are parsed via [parseJSONResponse], which unwraps the Subsonic JSON envelope
// and returns the inner response map.
//
// # Test Organization
//
// Each test file covers a logical group of Subsonic API endpoints:
//
//   - subsonic_system_test.go: ping, getLicense, getOpenSubsonicExtensions
//   - subsonic_browsing_test.go: getMusicFolders, getIndexes, getArtists, getMusicDirectory,
//     getArtist, getAlbum, getSong, getGenres
//   - subsonic_searching_test.go: search2, search3
//   - subsonic_album_lists_test.go: getAlbumList, getAlbumList2
//   - subsonic_playlists_test.go: createPlaylist, getPlaylist, getPlaylists,
//     updatePlaylist, deletePlaylist
//   - subsonic_media_annotation_test.go: star, unstar, getStarred, setRating, scrobble
//   - subsonic_media_retrieval_test.go: stream, download, getCoverArt, getAvatar,
//     getLyrics, getLyricsBySongId
//   - subsonic_bookmarks_test.go: createBookmark, getBookmarks, deleteBookmark,
//     savePlayQueue, getPlayQueue
//   - subsonic_radio_test.go: getInternetRadioStations, createInternetRadioStation,
//     updateInternetRadioStation, deleteInternetRadioStation
//   - subsonic_sharing_test.go: createShare, getShares, updateShare, deleteShare
//   - subsonic_users_test.go: getUser, getUsers
//   - subsonic_scan_test.go: getScanStatus, startScan
//   - subsonic_multiuser_test.go: multi-user isolation and permission enforcement
//   - subsonic_multilibrary_test.go: multi-library access control and data isolation
//
// Some test groups use Ginkgo's Ordered decorator to run tests sequentially within a block,
// allowing later tests to depend on state created by earlier ones (e.g., creating a playlist
// and then verifying it can be retrieved).
//
// # Running
//
// The e2e tests are included in the standard test suite and can be run with:
//
//	make test PKG=./server/e2e   # Run only e2e tests
//	make test                     # Run all tests including e2e
//	make test-race                # Run with race detector
//
// [Ginkgo]: https://onsi.github.io/ginkgo/
// [Gomega]: https://onsi.github.io/gomega/
// [storagetest]: /core/storage/storagetest
package e2e
