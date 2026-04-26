package e2e

import (
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Smart Playlists", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("String fields", func() {
		It("matches by exact title", func() {
			results := evaluateRule(`{"all":[{"is":{"title":"Something"}}]}`)
			Expect(results).To(ConsistOf("Something"))
		})

		It("matches by title contains", func() {
			results := evaluateRule(`{"all":[{"contains":{"title":"the"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "All Along the Watchtower", "We Are the Champions"))
		})

		It("matches by artist startsWith", func() {
			results := evaluateRule(`{"all":[{"startsWith":{"artist":"Led"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Black Dog"))
		})

		It("matches by title isNot", func() {
			results := evaluateRule(`{"all":[{"isNot":{"title":"Something"}},{"is":{"artist":"The Beatles"}}]}`)
			Expect(results).To(ConsistOf("Come Together"))
		})

		It("matches by artist endsWith", func() {
			results := evaluateRule(`{"all":[{"endsWith":{"artist":"Davis"}}]}`)
			Expect(results).To(ConsistOf("So What"))
		})
	})

	Describe("Numeric fields", func() {
		It("matches by year greater than", func() {
			results := evaluateRule(`{"all":[{"gt":{"year":1970}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Black Dog", "Bohemian Rhapsody", "We Are the Champions"))
		})

		It("matches by year less than", func() {
			results := evaluateRule(`{"all":[{"lt":{"year":1969}}]}`)
			Expect(results).To(ConsistOf("So What", "All Along the Watchtower"))
		})

		It("matches by BPM in range", func() {
			results := evaluateRule(`{"all":[{"inTheRange":{"bpm":[100,130]}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "All Along the Watchtower"))
		})
	})

	Describe("Boolean fields", func() {
		It("matches compilations", func() {
			results := evaluateRule(`{"all":[{"is":{"compilation":true}}]}`)
			Expect(results).To(ConsistOf("We Are the Champions"))
		})

		It("matches non-compilations", func() {
			results := evaluateRule(`{"all":[{"is":{"compilation":false}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog", "So What", "Bohemian Rhapsody", "All Along the Watchtower"))
		})
	})

	Describe("File type fields", func() {
		It("matches by filetype", func() {
			results := evaluateRule(`{"all":[{"is":{"filetype":"flac"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Black Dog"))
		})
	})

	Describe("Multi-valued tags", func() {
		It("matches tracks with Blues genre", func() {
			results := evaluateRule(`{"all":[{"is":{"genre":"Blues"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Black Dog", "All Along the Watchtower"))
		})

		It("excludes tracks with Rock genre", func() {
			results := evaluateRule(`{"all":[{"isNot":{"genre":"Rock"}}]}`)
			Expect(results).To(ConsistOf("So What"))
		})

		It("matches genre contains", func() {
			results := evaluateRule(`{"all":[{"contains":{"genre":"ol"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven"))
		})

		It("matches tracks with Pop genre", func() {
			results := evaluateRule(`{"all":[{"is":{"genre":"Pop"}}]}`)
			Expect(results).To(ConsistOf("We Are the Champions"))
		})

		It("matches genre startsWith", func() {
			results := evaluateRule(`{"all":[{"startsWith":{"genre":"Ro"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog",
				"Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})
	})

	Describe("Participants", func() {
		It("matches by exact composer", func() {
			results := evaluateRule(`{"all":[{"is":{"composer":"Harrison"}}]}`)
			Expect(results).To(ConsistOf("Something"))
		})

		It("matches by composer contains", func() {
			results := evaluateRule(`{"all":[{"contains":{"composer":"Plant"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Black Dog"))
		})

		It("matches by composer isNot", func() {
			results := evaluateRule(`{"all":[{"isNot":{"composer":"Freddie Mercury"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog", "So What", "All Along the Watchtower"))
		})

		It("matches by composer endsWith", func() {
			results := evaluateRule(`{"all":[{"endsWith":{"composer":"Mercury"}}]}`)
			Expect(results).To(ConsistOf("Bohemian Rhapsody", "We Are the Champions"))
		})
	})

	Describe("Annotations", func() {
		It("matches starred tracks", func() {
			results := evaluateRule(`{"all":[{"is":{"loved":true}}]}`)
			Expect(results).To(ConsistOf("Come Together", "So What"))
		})

		It("matches unstarred tracks", func() {
			results := evaluateRule(`{"all":[{"is":{"loved":false}}]}`)
			Expect(results).To(ConsistOf("Something", "Stairway To Heaven", "Black Dog", "Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})

		It("matches by rating greater than", func() {
			results := evaluateRule(`{"all":[{"gt":{"rating":3}}]}`)
			Expect(results).To(ConsistOf("Bohemian Rhapsody"))
		})

		It("matches by rating greater than or equal via inTheRange", func() {
			results := evaluateRule(`{"all":[{"inTheRange":{"rating":[3,5]}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Bohemian Rhapsody"))
		})

		It("matches by play count greater than", func() {
			results := evaluateRule(`{"all":[{"gt":{"playcount":5}}]}`)
			Expect(results).To(ConsistOf("Come Together"))
		})

		It("matches by play count greater than zero", func() {
			results := evaluateRule(`{"all":[{"gt":{"playcount":0}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Black Dog"))
		})
	})

	Describe("Negated string operators", func() {
		It("matches by title notContains", func() {
			results := evaluateRule(`{"all":[{"notContains":{"title":"the"}}]}`)
			Expect(results).To(ConsistOf("Something", "Stairway To Heaven", "Black Dog", "So What", "Bohemian Rhapsody"))
		})
	})

	Describe("Date/time fields", func() {
		It("matches dateAdded before a far-future date", func() {
			results := evaluateRule(`{"all":[{"before":{"dateadded":"2099-01-01"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog",
				"So What", "Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})

		It("matches lastPlayed inTheLast 1 day", func() {
			results := evaluateRule(`{"all":[{"inTheLast":{"lastplayed":1}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Black Dog"))
		})

		It("matches lastPlayed notInTheLast (far future)", func() {
			results := evaluateRule(`{"all":[{"notInTheLast":{"lastplayed":99999}}]}`)
			Expect(results).To(ConsistOf("Something", "Stairway To Heaven", "So What",
				"Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})

		It("matches dateLoved after a past date", func() {
			results := evaluateRule(`{"all":[{"after":{"dateloved":"2020-01-01"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "So What"))
		})

		It("matches dateRated after a past date", func() {
			results := evaluateRule(`{"all":[{"after":{"daterated":"2020-01-01"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Bohemian Rhapsody"))
		})

		It("matches dateAdded inTheLast 1 day", func() {
			results := evaluateRule(`{"all":[{"inTheLast":{"dateadded":1}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog",
				"So What", "Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})

		It("resolves recordingdate alias to the date column", func() {
			results := evaluateRule(`{"all":[{"is":{"recordingdate":"1959"}}]}`)
			Expect(results).To(ConsistOf("So What"))
		})
	})

	Describe("Logic operators", func() {
		It("matches with ALL (AND)", func() {
			results := evaluateRule(`{"all":[{"is":{"genre":"Blues"}},{"gt":{"bpm":130}}]}`)
			Expect(results).To(ConsistOf("Black Dog"))
		})

		It("matches with ANY (OR)", func() {
			results := evaluateRule(`{"any":[{"is":{"genre":"Jazz"}},{"is":{"compilation":true}}]}`)
			Expect(results).To(ConsistOf("So What", "We Are the Champions"))
		})

		It("matches nested all/any", func() {
			results := evaluateRule(`{"all":[{"any":[{"is":{"genre":"Blues"}},{"is":{"genre":"Jazz"}}]},{"gt":{"year":1960}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Black Dog", "All Along the Watchtower"))
		})
	})

	Describe("Sorting and limits", func() {
		It("returns tracks sorted by year descending with limit", func() {
			results := evaluateRuleOrdered(`{"all":[{"gt":{"year":0}}],"sort":"year","order":"desc","limit":2}`)
			Expect(results).To(Equal([]string{"We Are the Champions", "Bohemian Rhapsody"}))
		})

		It("returns tracks sorted by title ascending", func() {
			results := evaluateRuleOrdered(`{"all":[{"is":{"genre":"Blues"}}],"sort":"title","order":"asc"}`)
			Expect(results).To(Equal([]string{"All Along the Watchtower", "Black Dog", "Come Together"}))
		})
	})

	Describe("Combined real-world patterns", func() {
		It("matches genre filter with exclusion and year range", func() {
			results := evaluateRuleOrdered(`{
				"all":[
					{"any":[
						{"is":{"genre":"Blues"}},
						{"is":{"genre":"Folk"}}
					]},
					{"isNot":{"genre":"Jazz"}},
					{"gt":{"year":1965}}
				],
				"sort":"-year,title"
			}`)
			Expect(results).To(Equal([]string{"Black Dog", "Stairway To Heaven", "Come Together", "All Along the Watchtower"}))
		})
	})

	Describe("Playlist operators", func() {
		It("matches tracks in a public regular playlist", func() {
			refID := createPublicPlaylist(adminUser, "Come Together", "So What")
			results := evaluateRuleAs(regularUser, `{"all":[{"inPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "So What"))
		})

		It("matches tracks not in a public regular playlist", func() {
			refID := createPublicPlaylist(adminUser, "Come Together", "So What")
			results := evaluateRuleAs(regularUser, `{"all":[{"notInPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Something", "Stairway To Heaven", "Black Dog",
				"Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))
		})

		It("recursively refreshes a referenced smart playlist owned by the same user", func() {
			smartBID := createPublicSmartPlaylist(adminUser, `{"all":[{"is":{"genre":"Jazz"}}]}`)
			results := evaluateRuleAs(adminUser, `{"all":[{"inPlaylist":{"id":"`+smartBID+`"}}]}`)
			Expect(results).To(ConsistOf("So What"))
		})

		It("does not refresh a referenced smart playlist owned by another user", func() {
			smartBID := createPublicSmartPlaylist(regularUser, `{"all":[{"is":{"genre":"Jazz"}}]}`)
			results := evaluateRuleAs(adminUser, `{"all":[{"inPlaylist":{"id":"`+smartBID+`"}}]}`)
			Expect(results).To(BeEmpty())
		})

		It("does not refresh a playlist or its children when an admin views another user's smart playlist", func() {
			smartBID := createPrivateSmartPlaylist(adminUser, `{"all":[{"is":{"genre":"Jazz"}}]}`)
			smartAID := createPublicSmartPlaylist(regularUser, `{"all":[{"inPlaylist":{"id":"`+smartBID+`"}}]}`)

			loadedA, err := ds.Playlist(ctx).GetWithTracks(smartAID, true, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(loadedA.Tracks).To(BeEmpty())
			Expect(loadedA.EvaluatedAt).To(BeNil())

			loadedB, err := ds.Playlist(ctx).Get(smartBID)
			Expect(err).ToNot(HaveOccurred())
			Expect(loadedB.EvaluatedAt).To(BeNil())
		})

		It("matches tracks from a private playlist owned by the same user", func() {
			refID := createPrivatePlaylist(regularUser, "Come Together", "So What")
			results := evaluateRuleAs(regularUser, `{"all":[{"inPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "So What"))
		})

		It("allows admin-owned smart playlists to reference private playlists owned by other users", func() {
			refID := createPrivatePlaylist(regularUser, "Bohemian Rhapsody")
			results := evaluateRuleAs(adminUser, `{"all":[{"inPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Bohemian Rhapsody"))
		})

		It("does not match tracks from a private playlist owned by another regular user", func() {
			refID := createPrivatePlaylist(adminUser, "Come Together", "So What")
			results := evaluateRuleAs(regularUser, `{"all":[{"inPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(BeEmpty())
		})

		It("warns when a referenced playlist is inaccessible to the smart playlist owner", func() {
			hook, cleanup := tests.LogHook()
			defer cleanup()

			refID := createPrivatePlaylist(adminUser, "Come Together")
			results := evaluateRuleAs(regularUser, `{"all":[{"notInPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "Something", "Stairway To Heaven", "Black Dog",
				"So What", "Bohemian Rhapsody", "All Along the Watchtower", "We Are the Champions"))

			Expect(hook.LastEntry()).ToNot(BeNil())
			Expect(hook.LastEntry().Level).To(Equal(logrus.WarnLevel))
			Expect(hook.LastEntry().Message).To(Equal("Referenced playlist is not accessible to smart playlist owner"))
			Expect(hook.LastEntry().Data).To(HaveKeyWithValue("childId", refID))
		})

		It("matches tracks in a public playlist owned by another user", func() {
			refID := createPublicPlaylist(adminUser, "Bohemian Rhapsody")
			results := evaluateRuleAs(regularUser, `{"all":[{"inPlaylist":{"id":"`+refID+`"}}]}`)
			Expect(results).To(ConsistOf("Bohemian Rhapsody"))
		})

	})
})
