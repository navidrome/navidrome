package persistence

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {

	Context("Core Functionality", func() {
		Describe("GetIndexKey", func() {
			// Note: OrderArtistName should never be empty, so we don't need to test for that
			r := artistRepository{indexGroups: utils.ParseIndexGroups(conf.Server.IndexGroups)}

			DescribeTable("returns correct index key based on PreferSortTags setting",
				func(preferSortTags bool, sortArtistName, orderArtistName, expectedKey string) {
					DeferCleanup(configtest.SetupConfig())
					conf.Server.PreferSortTags = preferSortTags
					a := model.Artist{SortArtistName: sortArtistName, OrderArtistName: orderArtistName, Name: "Test"}
					idx := GetIndexKey(&r, a)
					Expect(idx).To(Equal(expectedKey))
				},
				Entry("PreferSortTags=false, SortArtistName empty -> uses OrderArtistName", false, "", "Bar", "B"),
				Entry("PreferSortTags=false, SortArtistName not empty -> still uses OrderArtistName", false, "Foo", "Bar", "B"),
				Entry("PreferSortTags=true, SortArtistName not empty -> uses SortArtistName", true, "Foo", "Bar", "F"),
				Entry("PreferSortTags=true, SortArtistName empty -> falls back to OrderArtistName", true, "", "Bar", "B"),
			)
		})

		Describe("roleFilter", func() {
			DescribeTable("validates roles and returns appropriate SQL expressions",
				func(role string, shouldBeValid bool) {
					result := roleFilter("", role)
					if shouldBeValid {
						expectedExpr := squirrel.Expr("EXISTS (SELECT 1 FROM library_artist WHERE library_artist.artist_id = artist.id AND JSON_EXTRACT(library_artist.stats, '$." + role + ".m') IS NOT NULL)")
						Expect(result).To(Equal(expectedExpr))
					} else {
						expectedInvalid := squirrel.Eq{"1": 2}
						Expect(result).To(Equal(expectedInvalid))
					}
				},
				// Valid roles from model.AllRoles
				Entry("artist role", "artist", true),
				Entry("albumartist role", "albumartist", true),
				Entry("composer role", "composer", true),
				Entry("conductor role", "conductor", true),
				Entry("lyricist role", "lyricist", true),
				Entry("arranger role", "arranger", true),
				Entry("producer role", "producer", true),
				Entry("director role", "director", true),
				Entry("engineer role", "engineer", true),
				Entry("mixer role", "mixer", true),
				Entry("remixer role", "remixer", true),
				Entry("djmixer role", "djmixer", true),
				Entry("performer role", "performer", true),
				Entry("maincredit role", "maincredit", true),
				// Invalid roles
				Entry("invalid role - wizard", "wizard", false),
				Entry("invalid role - songanddanceman", "songanddanceman", false),
				Entry("empty string", "", false),
				Entry("SQL injection attempt", "artist') SELECT LIKE(CHAR(65,66,67,68,69,70,71),UPPER(HEX(RANDOMBLOB(500000000/2))))--", false),
			)

			It("handles non-string input types", func() {
				expectedInvalid := squirrel.Eq{"1": 2}
				Expect(roleFilter("", 123)).To(Equal(expectedInvalid))
				Expect(roleFilter("", nil)).To(Equal(expectedInvalid))
				Expect(roleFilter("", []string{"artist"})).To(Equal(expectedInvalid))
			})
		})

		Describe("dbArtist mapping", func() {
			var (
				artist *model.Artist
				dba    *dbArtist
			)

			BeforeEach(func() {
				artist = &model.Artist{ID: "1", Name: "Eddie Van Halen", SortArtistName: "Van Halen, Eddie"}
				dba = &dbArtist{Artist: artist}
			})

			Describe("PostScan", func() {
				It("parses stats and similar artists correctly", func() {
					stats := map[string]map[string]map[string]int64{
						"1": {
							"total":    {"s": 1000, "m": 10, "a": 2},
							"composer": {"s": 500, "m": 5, "a": 1},
						},
					}
					statsJSON, _ := json.Marshal(stats)
					dba.LibraryStatsJSON = string(statsJSON)
					dba.SimilarArtists = `[{"id":"2","Name":"AC/DC"},{"name":"Test;With:Sep,Chars"}]`

					err := dba.PostScan()
					Expect(err).ToNot(HaveOccurred())
					Expect(dba.Artist.Size).To(Equal(int64(1000)))
					Expect(dba.Artist.SongCount).To(Equal(10))
					Expect(dba.Artist.AlbumCount).To(Equal(2))
					Expect(dba.Artist.Stats).To(HaveLen(1))
					Expect(dba.Artist.Stats[model.RoleFromString("composer")].Size).To(Equal(int64(500)))
					Expect(dba.Artist.Stats[model.RoleFromString("composer")].SongCount).To(Equal(5))
					Expect(dba.Artist.Stats[model.RoleFromString("composer")].AlbumCount).To(Equal(1))
					Expect(dba.Artist.SimilarArtists).To(HaveLen(2))
					Expect(dba.Artist.SimilarArtists[0].ID).To(Equal("2"))
					Expect(dba.Artist.SimilarArtists[0].Name).To(Equal("AC/DC"))
					Expect(dba.Artist.SimilarArtists[1].ID).To(BeEmpty())
					Expect(dba.Artist.SimilarArtists[1].Name).To(Equal("Test;With:Sep,Chars"))
				})
			})

			Describe("PostMapArgs", func() {
				It("maps empty similar artists correctly", func() {
					m := make(map[string]any)
					err := dba.PostMapArgs(m)
					Expect(err).ToNot(HaveOccurred())
					Expect(m).To(HaveKeyWithValue("similar_artists", "[]"))
				})

				It("maps similar artists and full text correctly", func() {
					artist.SimilarArtists = []model.Artist{
						{ID: "2", Name: "AC/DC"},
						{Name: "Test;With:Sep,Chars"},
					}
					m := make(map[string]any)
					err := dba.PostMapArgs(m)
					Expect(err).ToNot(HaveOccurred())
					Expect(m).To(HaveKeyWithValue("similar_artists", `[{"id":"2","name":"AC/DC"},{"name":"Test;With:Sep,Chars"}]`))
					Expect(m).To(HaveKeyWithValue("full_text", " eddie halen van"))
				})

				It("does not override empty sort_artist_name and mbz_artist_id", func() {
					m := map[string]any{
						"sort_artist_name": "",
						"mbz_artist_id":    "",
					}
					err := dba.PostMapArgs(m)
					Expect(err).ToNot(HaveOccurred())
					Expect(m).ToNot(HaveKey("sort_artist_name"))
					Expect(m).ToNot(HaveKey("mbz_artist_id"))
				})
			})
		})
	})

	Context("Admin User Operations", func() {
		var repo model.ArtistRepository

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, adminUser)
			repo = NewArtistRepository(ctx, GetDBXBuilder())
		})

		Describe("Basic Operations", func() {
			Describe("Count", func() {
				It("returns the number of artists in the DB", func() {
					Expect(repo.CountAll()).To(Equal(int64(2)))
				})
			})

			Describe("Exists", func() {
				It("returns true for an artist that is in the DB", func() {
					Expect(repo.Exists("3")).To(BeTrue())
				})
				It("returns false for an artist that is NOT in the DB", func() {
					Expect(repo.Exists("666")).To(BeFalse())
				})
			})

			Describe("Get", func() {
				It("retrieves existing artist data", func() {
					artist, err := repo.Get("2")
					Expect(err).ToNot(HaveOccurred())
					Expect(artist.Name).To(Equal(artistKraftwerk.Name))
				})
			})
		})

		Describe("GetIndex", func() {
			When("PreferSortTags is true", func() {
				BeforeEach(func() {
					conf.Server.PreferSortTags = true
				})
				It("returns the index when PreferSortTags is true and SortArtistName is not empty", func() {
					// Set SortArtistName to "Foo" for Beatles
					artistBeatles.SortArtistName = "Foo"
					er := repo.Put(&artistBeatles)
					Expect(er).To(BeNil())

					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))
					Expect(idx[0].ID).To(Equal("F"))
					Expect(idx[0].Artists).To(HaveLen(1))
					Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
					Expect(idx[1].ID).To(Equal("K"))
					Expect(idx[1].Artists).To(HaveLen(1))
					Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))

					// Restore the original value
					artistBeatles.SortArtistName = ""
					er = repo.Put(&artistBeatles)
					Expect(er).To(BeNil())
				})

				// BFR Empty SortArtistName is not saved in the DB anymore
				XIt("returns the index when PreferSortTags is true and SortArtistName is empty", func() {
					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))
					Expect(idx[0].ID).To(Equal("B"))
					Expect(idx[0].Artists).To(HaveLen(1))
					Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
					Expect(idx[1].ID).To(Equal("K"))
					Expect(idx[1].Artists).To(HaveLen(1))
					Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))
				})
			})

			When("PreferSortTags is false", func() {
				BeforeEach(func() {
					conf.Server.PreferSortTags = false
				})
				It("returns the index when SortArtistName is NOT empty", func() {
					// Set SortArtistName to "Foo" for Beatles
					artistBeatles.SortArtistName = "Foo"
					er := repo.Put(&artistBeatles)
					Expect(er).To(BeNil())

					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))
					Expect(idx[0].ID).To(Equal("B"))
					Expect(idx[0].Artists).To(HaveLen(1))
					Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
					Expect(idx[1].ID).To(Equal("K"))
					Expect(idx[1].Artists).To(HaveLen(1))
					Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))

					// Restore the original value
					artistBeatles.SortArtistName = ""
					er = repo.Put(&artistBeatles)
					Expect(er).To(BeNil())
				})

				It("returns the index when SortArtistName is empty", func() {
					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))
					Expect(idx[0].ID).To(Equal("B"))
					Expect(idx[0].Artists).To(HaveLen(1))
					Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
					Expect(idx[1].ID).To(Equal("K"))
					Expect(idx[1].Artists).To(HaveLen(1))
					Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))
				})
			})

			When("filtering by role", func() {
				var raw *artistRepository

				BeforeEach(func() {
					raw = repo.(*artistRepository)
					// Add stats to library_artist table since stats are now stored per-library
					composerStats := `{"composer": {"s": 1000, "m": 5, "a": 2}}`
					producerStats := `{"producer": {"s": 500, "m": 3, "a": 1}}`

					// Set Beatles as composer in library 1
					_, err := raw.executeSQL(squirrel.Insert("library_artist").
						Columns("library_id", "artist_id", "stats").
						Values(1, artistBeatles.ID, composerStats).
						Suffix("ON CONFLICT(library_id, artist_id) DO UPDATE SET stats = excluded.stats"))
					Expect(err).ToNot(HaveOccurred())

					// Set Kraftwerk as producer in library 1
					_, err = raw.executeSQL(squirrel.Insert("library_artist").
						Columns("library_id", "artist_id", "stats").
						Values(1, artistKraftwerk.ID, producerStats).
						Suffix("ON CONFLICT(library_id, artist_id) DO UPDATE SET stats = excluded.stats"))
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					// Clean up stats from library_artist table
					_, _ = raw.executeSQL(squirrel.Update("library_artist").
						Set("stats", "{}").
						Where(squirrel.Eq{"artist_id": artistBeatles.ID, "library_id": 1}))
					_, _ = raw.executeSQL(squirrel.Update("library_artist").
						Set("stats", "{}").
						Where(squirrel.Eq{"artist_id": artistKraftwerk.ID, "library_id": 1}))
				})

				It("returns only artists with the specified role", func() {
					idx, err := repo.GetIndex(false, []int{1}, model.RoleComposer)
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(1))
					Expect(idx[0].ID).To(Equal("B"))
					Expect(idx[0].Artists).To(HaveLen(1))
					Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
				})

				It("returns artists with any of the specified roles", func() {
					idx, err := repo.GetIndex(false, []int{1}, model.RoleComposer, model.RoleProducer)
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))

					// Find Beatles and Kraftwerk in the results
					var beatlesFound, kraftwerkFound bool
					for _, index := range idx {
						for _, artist := range index.Artists {
							if artist.Name == artistBeatles.Name {
								beatlesFound = true
							}
							if artist.Name == artistKraftwerk.Name {
								kraftwerkFound = true
							}
						}
					}
					Expect(beatlesFound).To(BeTrue())
					Expect(kraftwerkFound).To(BeTrue())
				})

				It("returns empty index when no artists have the specified role", func() {
					idx, err := repo.GetIndex(false, []int{1}, model.RoleDirector)
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(0))
				})
			})

			When("validating library IDs", func() {
				It("returns nil when no library IDs are provided", func() {
					idx, err := repo.GetIndex(false, []int{})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(BeNil())
				})

				It("returns artists when library IDs are provided (admin user sees all content)", func() {
					// Admin users can see all content when valid library IDs are provided
					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(2))

					// With non-existent library ID, admin users see no content because no artists are associated with that library
					idx, err = repo.GetIndex(false, []int{999})
					Expect(err).ToNot(HaveOccurred())
					Expect(idx).To(HaveLen(0)) // Even admin users need valid library associations
				})
			})
		})

		Describe("MBID Search", func() {
			var artistWithMBID model.Artist
			var raw *artistRepository

			BeforeEach(func() {
				raw = repo.(*artistRepository)
				// Create a test artist with MBID
				artistWithMBID = model.Artist{
					ID:          "test-mbid-artist",
					Name:        "Test MBID Artist",
					MbzArtistID: "550e8400-e29b-41d4-a716-446655440010", // Valid UUID v4
				}

				// Insert the test artist into the database with proper library association
				err := createArtistWithLibrary(repo, &artistWithMBID, 1)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				// Clean up test data using direct SQL
				_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": artistWithMBID.ID}))
			})

			It("finds artist by mbz_artist_id", func() {
				results, err := repo.Search("550e8400-e29b-41d4-a716-446655440010", 0, 10, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].ID).To(Equal("test-mbid-artist"))
				Expect(results[0].Name).To(Equal("Test MBID Artist"))
			})

			It("returns empty result when MBID is not found", func() {
				results, err := repo.Search("550e8400-e29b-41d4-a716-446655440099", 0, 10, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("handles includeMissing parameter for MBID search", func() {
				// Create a missing artist with MBID
				missingArtist := model.Artist{
					ID:          "test-missing-mbid-artist",
					Name:        "Test Missing MBID Artist",
					MbzArtistID: "550e8400-e29b-41d4-a716-446655440012",
					Missing:     true,
				}

				err := createArtistWithLibrary(repo, &missingArtist, 1)
				Expect(err).ToNot(HaveOccurred())

				// Should not find missing artist when includeMissing is false
				results, err := repo.Search("550e8400-e29b-41d4-a716-446655440012", 0, 10, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())

				// Should find missing artist when includeMissing is true
				results, err = repo.Search("550e8400-e29b-41d4-a716-446655440012", 0, 10, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].ID).To(Equal("test-missing-mbid-artist"))

				// Clean up
				_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": missingArtist.ID}))
			})
		})

		Describe("Admin User Library Access", func() {
			It("sees all artists regardless of library permissions", func() {
				count, err := repo.CountAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(2)))

				artists, err := repo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(artists).To(HaveLen(2))

				exists, err := repo.Exists(artistBeatles.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
	})

	Context("Regular User Operations", func() {
		var restrictedRepo model.ArtistRepository
		var unauthorizedUser model.User

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			// Create a user without access to any libraries
			unauthorizedUser = model.User{ID: "restricted_user", UserName: "restricted", Name: "Restricted User", Email: "restricted@test.com", IsAdmin: false}

			// Create repository context for the unauthorized user
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, unauthorizedUser)
			restrictedRepo = NewArtistRepository(ctx, GetDBXBuilder())
		})

		Describe("Library Access Restrictions", func() {
			It("CountAll returns 0 for users without library access", func() {
				count, err := restrictedRepo.CountAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(0)))
			})

			It("GetAll returns empty list for users without library access", func() {
				artists, err := restrictedRepo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(artists).To(BeEmpty())
			})

			It("Exists returns false for existing artists when user has no library access", func() {
				// These artists exist in the DB but the user has no access to them
				exists, err := restrictedRepo.Exists(artistBeatles.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())

				exists, err = restrictedRepo.Exists(artistKraftwerk.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})

			It("Get returns ErrNotFound for existing artists when user has no library access", func() {
				_, err := restrictedRepo.Get(artistBeatles.ID)
				Expect(err).To(Equal(model.ErrNotFound))

				_, err = restrictedRepo.Get(artistKraftwerk.ID)
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("Search returns empty results for users without library access", func() {
				results, err := restrictedRepo.Search("Beatles", 0, 10, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())

				results, err = restrictedRepo.Search("Kraftwerk", 0, 10, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("GetIndex returns empty index for users without library access", func() {
				idx, err := restrictedRepo.GetIndex(false, []int{1})
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(0))
			})
		})

		Context("when user gains library access", func() {
			BeforeEach(func() {
				// Give the user access to library 1
				ur := NewUserRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())

				// First create the user if not exists
				err := ur.Put(&unauthorizedUser)
				Expect(err).ToNot(HaveOccurred())

				// Then add library access
				err = ur.SetUserLibraries(unauthorizedUser.ID, []int{1})
				Expect(err).ToNot(HaveOccurred())

				// Update the user object with the libraries to simulate middleware behavior
				libraries, err := ur.GetUserLibraries(unauthorizedUser.ID)
				Expect(err).ToNot(HaveOccurred())
				unauthorizedUser.Libraries = libraries

				// Recreate repository context with updated user
				ctx := log.NewContext(context.TODO())
				ctx = request.WithUser(ctx, unauthorizedUser)
				restrictedRepo = NewArtistRepository(ctx, GetDBXBuilder())
			})

			AfterEach(func() {
				// Clean up: remove the user's library access
				ur := NewUserRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				_ = ur.SetUserLibraries(unauthorizedUser.ID, []int{})
			})

			It("CountAll returns correct count after gaining access", func() {
				count, err := restrictedRepo.CountAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(2))) // Beatles and Kraftwerk
			})

			It("GetAll returns artists after gaining access", func() {
				artists, err := restrictedRepo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(artists).To(HaveLen(2))

				var names []string
				for _, artist := range artists {
					names = append(names, artist.Name)
				}
				Expect(names).To(ContainElements("The Beatles", "Kraftwerk"))
			})

			It("Exists returns true for accessible artists", func() {
				exists, err := restrictedRepo.Exists(artistBeatles.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())

				exists, err = restrictedRepo.Exists(artistKraftwerk.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})

			It("GetIndex returns artists with proper library filtering", func() {
				// With valid library access, should see artists
				idx, err := restrictedRepo.GetIndex(false, []int{1})
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(2))

				// With non-existent library ID, should see nothing (non-admin user)
				idx, err = restrictedRepo.GetIndex(false, []int{999})
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(0))
			})
		})
	})

	Context("Permission-Based Behavior Comparison", func() {
		Describe("Missing Artist Visibility", func() {
			var repo model.ArtistRepository
			var raw *artistRepository
			var missing model.Artist

			insertMissing := func() {
				missing = model.Artist{ID: "m1", Name: "Missing", OrderArtistName: "missing"}
				Expect(repo.Put(&missing)).To(Succeed())
				raw = repo.(*artistRepository)
				_, err := raw.executeSQL(squirrel.Update(raw.tableName).Set("missing", true).Where(squirrel.Eq{"id": missing.ID}))
				Expect(err).ToNot(HaveOccurred())

				// Add missing artist to library 1 so it can be found by library filtering
				lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				err = lr.AddArtist(1, missing.ID)
				Expect(err).ToNot(HaveOccurred())

				// Ensure the test user exists and has library access
				ur := NewUserRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				currentUser, ok := request.UserFrom(repo.(*artistRepository).ctx)
				if ok {
					// Create the user if it doesn't exist with default values if missing
					testUser := model.User{
						ID:       currentUser.ID,
						UserName: currentUser.UserName,
						Name:     currentUser.Name,
						Email:    currentUser.Email,
						IsAdmin:  currentUser.IsAdmin,
					}
					// Provide defaults for missing fields
					if testUser.UserName == "" {
						testUser.UserName = testUser.ID
					}
					if testUser.Name == "" {
						testUser.Name = testUser.ID
					}
					if testUser.Email == "" {
						testUser.Email = testUser.ID + "@test.com"
					}

					// Try to put the user (will fail silently if already exists)
					_ = ur.Put(&testUser)

					// Add library association using SetUserLibraries
					err = ur.SetUserLibraries(currentUser.ID, []int{1})
					// Ignore error if user already has these libraries or other conflicts
					if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") && !strings.Contains(err.Error(), "duplicate key") {
						Expect(err).ToNot(HaveOccurred())
					}
				}
			}

			removeMissing := func() {
				if raw != nil {
					_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": missing.ID}))
				}
			}

			Context("regular user", func() {
				BeforeEach(func() {
					DeferCleanup(configtest.SetupConfig())
					// Create user with library access (simulating middleware behavior)
					regularUserWithLibs := model.User{
						ID:      "u1",
						IsAdmin: false,
						Libraries: model.Libraries{
							{ID: 1, Name: "Test Library", Path: "/test"},
						},
					}
					ctx := log.NewContext(context.TODO())
					ctx = request.WithUser(ctx, regularUserWithLibs)
					repo = NewArtistRepository(ctx, GetDBXBuilder())
					insertMissing()
				})

				AfterEach(func() { removeMissing() })

				It("does not return missing artist in GetAll", func() {
					artists, err := repo.GetAll(model.QueryOptions{Filters: squirrel.Eq{"artist.missing": false}})
					Expect(err).ToNot(HaveOccurred())
					Expect(artists).To(HaveLen(2))
				})

				It("does not return missing artist in Search", func() {
					res, err := repo.Search("missing", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(res).To(BeEmpty())
				})

				It("does not return missing artist in GetIndex", func() {
					idx, err := repo.GetIndex(false, []int{1})
					Expect(err).ToNot(HaveOccurred())
					// Only 2 artists should be present
					total := 0
					for _, ix := range idx {
						total += len(ix.Artists)
					}
					Expect(total).To(Equal(2))
				})
			})

			Context("admin user", func() {
				BeforeEach(func() {
					DeferCleanup(configtest.SetupConfig())
					ctx := log.NewContext(context.TODO())
					ctx = request.WithUser(ctx, model.User{ID: "admin", IsAdmin: true})
					repo = NewArtistRepository(ctx, GetDBXBuilder())
					insertMissing()
				})

				AfterEach(func() { removeMissing() })

				It("returns missing artist in GetAll", func() {
					artists, err := repo.GetAll()
					Expect(err).ToNot(HaveOccurred())
					Expect(artists).To(HaveLen(3))
				})

				It("returns missing artist in Search", func() {
					res, err := repo.Search("missing", 0, 10, true)
					Expect(err).ToNot(HaveOccurred())
					Expect(res).To(HaveLen(1))
				})

				It("returns missing artist in GetIndex when included", func() {
					idx, err := repo.GetIndex(true, []int{1})
					Expect(err).ToNot(HaveOccurred())
					total := 0
					for _, ix := range idx {
						total += len(ix.Artists)
					}
					Expect(total).To(Equal(3))
				})
			})
		})

		Describe("Library Filtering", func() {
			var restrictedUser model.User
			var restrictedRepo model.ArtistRepository
			var adminRepo model.ArtistRepository
			var lib2 model.Library

			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())

				// Set up admin repo
				ctx := log.NewContext(context.TODO())
				ctx = request.WithUser(ctx, adminUser)
				adminRepo = NewArtistRepository(ctx, GetDBXBuilder())

				// Create library for testing access restrictions
				lib2 = model.Library{ID: 0, Name: "Artist Test Library", Path: "/artist/test/lib"}
				lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				err := lr.Put(&lib2)
				Expect(err).ToNot(HaveOccurred())

				// Create a user with access to only library 1
				restrictedUser = model.User{
					ID:      "search_user",
					IsAdmin: false,
					Libraries: model.Libraries{
						{ID: 1, Name: "Library 1", Path: "/lib1"},
					},
				}

				// Create repository context for the restricted user
				ctx = log.NewContext(context.TODO())
				ctx = request.WithUser(ctx, restrictedUser)
				restrictedRepo = NewArtistRepository(ctx, GetDBXBuilder())

				// Ensure both test artists are associated with library 1
				err = lr.AddArtist(1, artistBeatles.ID)
				Expect(err).ToNot(HaveOccurred())
				err = lr.AddArtist(1, artistKraftwerk.ID)
				Expect(err).ToNot(HaveOccurred())

				// Create the restricted user in the database
				ur := NewUserRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				err = ur.Put(&restrictedUser)
				Expect(err).ToNot(HaveOccurred())
				err = ur.SetUserLibraries(restrictedUser.ID, []int{1})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				// Clean up library 2
				lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
				_ = lr.(*libraryRepository).delete(squirrel.Eq{"id": lib2.ID})
			})

			Context("MBID Search", func() {
				var artistWithMBID model.Artist

				BeforeEach(func() {
					artistWithMBID = model.Artist{
						ID:          "search-mbid-artist",
						Name:        "Search MBID Artist",
						MbzArtistID: "f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387",
					}
					err := createArtistWithLibrary(adminRepo, &artistWithMBID, 1)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					raw := adminRepo.(*artistRepository)
					_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": artistWithMBID.ID}))
				})

				It("allows admin to find artist by MBID regardless of library", func() {
					results, err := adminRepo.Search("f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(1))
					Expect(results[0].ID).To(Equal("search-mbid-artist"))
				})

				It("allows restricted user to find artist by MBID when in accessible library", func() {
					results, err := restrictedRepo.Search("f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(1))
					Expect(results[0].ID).To(Equal("search-mbid-artist"))
				})

				It("prevents restricted user from finding artist by MBID when not in accessible library", func() {
					// Create an artist in library 2 (not accessible to restricted user)
					inaccessibleArtist := model.Artist{
						ID:          "inaccessible-mbid-artist",
						Name:        "Inaccessible MBID Artist",
						MbzArtistID: "a74b1b7f-71a5-4011-9441-d0b5e4122711",
					}
					err := adminRepo.Put(&inaccessibleArtist)
					Expect(err).ToNot(HaveOccurred())

					// Add to library 2 (not accessible to restricted user)
					lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
					err = lr.AddArtist(lib2.ID, inaccessibleArtist.ID)
					Expect(err).ToNot(HaveOccurred())

					// Restricted user should not find this artist
					results, err := restrictedRepo.Search("a74b1b7f-71a5-4011-9441-d0b5e4122711", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(BeEmpty())

					// Clean up
					raw := adminRepo.(*artistRepository)
					_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": inaccessibleArtist.ID}))
				})
			})

			Context("Text Search", func() {
				It("allows admin to find artists by name regardless of library", func() {
					results, err := adminRepo.Search("Beatles", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(1))
					Expect(results[0].Name).To(Equal("The Beatles"))
				})

				It("correctly prevents restricted user from finding artists by name when not in accessible library", func() {
					// Create an artist in library 2 (not accessible to restricted user)
					inaccessibleArtist := model.Artist{
						ID:   "inaccessible-text-artist",
						Name: "Unique Search Name Artist",
					}
					err := adminRepo.Put(&inaccessibleArtist)
					Expect(err).ToNot(HaveOccurred())

					// Add to library 2 (not accessible to restricted user)
					lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
					err = lr.AddArtist(lib2.ID, inaccessibleArtist.ID)
					Expect(err).ToNot(HaveOccurred())

					// Restricted user should not find this artist
					results, err := restrictedRepo.Search("Unique Search Name", 0, 10, false)
					Expect(err).ToNot(HaveOccurred())

					// Text search correctly respects library filtering
					Expect(results).To(BeEmpty(), "Text search should respect library filtering")

					// Clean up
					raw := adminRepo.(*artistRepository)
					_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": inaccessibleArtist.ID}))
				})
			})
		})
	})
})

// Helper function to create an artist with proper library association.
// This ensures test artists always have library_artist associations to avoid orphaned artists in tests.
func createArtistWithLibrary(repo model.ArtistRepository, artist *model.Artist, libraryID int) error {
	err := repo.Put(artist)
	if err != nil {
		return err
	}

	// Add the artist to the specified library
	lr := NewLibraryRepository(request.WithUser(log.NewContext(context.TODO()), adminUser), GetDBXBuilder())
	return lr.AddArtist(libraryID, artist.ID)
}
