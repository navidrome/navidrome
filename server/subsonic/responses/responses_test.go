//go:build unix

// TODO Fix snapshot tests in Windows
// Response Snapshot tests. Only run in Linux and macOS, as they fail in Windows
// Probably because of EOL char differences
package responses_test

import (
	"encoding/json"
	"encoding/xml"
	"time"

	"github.com/navidrome/navidrome/consts"
	. "github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Responses", func() {
	var response *Subsonic
	BeforeEach(func() {
		response = &Subsonic{
			Status:        StatusOK,
			Version:       "1.8.0",
			Type:          consts.AppName,
			ServerVersion: "v0.0.0",
			OpenSubsonic:  true,
		}
	})

	Describe("EmptyResponse", func() {
		It("should match .XML", func() {
			Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
		})
		It("should match .JSON", func() {
			Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
		})
	})

	Describe("License", func() {
		BeforeEach(func() {
			response.License = &License{Valid: true}
		})
		It("should match .XML", func() {
			Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
		})
		It("should match .JSON", func() {
			Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
		})
	})

	Describe("MusicFolders", func() {
		BeforeEach(func() {
			response.MusicFolders = &MusicFolders{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				folders := make([]MusicFolder, 2)
				folders[0] = MusicFolder{Id: 111, Name: "aaa"}
				folders[1] = MusicFolder{Id: 222, Name: "bbb"}
				response.MusicFolders.Folders = folders
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Indexes", func() {
		BeforeEach(func() {
			response.Indexes = &Indexes{LastModified: 1, IgnoredArticles: "A"}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				artists := make([]Artist, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				artists[0] = Artist{
					Id:             "111",
					Name:           "aaa",
					Starred:        &t,
					UserRating:     3,
					AlbumCount:     2,
					ArtistImageUrl: "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png",
				}
				index := make([]Index, 1)
				index[0] = Index{Name: "A", Artists: artists}
				response.Indexes.Index = index
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Artist", func() {
		BeforeEach(func() {
			response.Artist = &Artists{LastModified: 1, IgnoredArticles: "A"}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				artists := make([]ArtistID3, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				artists[0] = ArtistID3{
					Id:             "111",
					Name:           "aaa",
					Starred:        &t,
					UserRating:     3,
					AlbumCount:     2,
					ArtistImageUrl: "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png",
				}
				index := make([]IndexID3, 1)
				index[0] = IndexID3{Name: "A", Artists: artists}
				response.Artist.Index = index
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data and MBID and Sort Name", func() {
			BeforeEach(func() {
				artists := make([]ArtistID3, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				artists[0] = ArtistID3{
					Id:             "111",
					Name:           "aaa",
					Starred:        &t,
					UserRating:     3,
					AlbumCount:     2,
					ArtistImageUrl: "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png",
					MusicBrainzId:  "1234",
					SortName:       "sort name",
				}
				index := make([]IndexID3, 1)
				index[0] = IndexID3{Name: "A", Artists: artists}
				response.Artist.Index = index
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Child", func() {
		Context("without data", func() {
			BeforeEach(func() {
				response.Directory = &Directory{Child: []Child{{Id: "1"}}}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
		Context("with data", func() {
			BeforeEach(func() {
				response.Directory = &Directory{Id: "1", Name: "N"}
				child := make([]Child, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				child[0] = Child{
					Id: "1", IsDir: true, Title: "title", Album: "album", Artist: "artist", Track: 1,
					Year: 1985, Genre: "Rock", CoverArt: "1", Size: 8421341, ContentType: "audio/flac",
					Suffix: "flac", TranscodedContentType: "audio/mpeg", TranscodedSuffix: "mp3",
					Duration: 146, BitRate: 320, Starred: &t, Genres: []ItemGenre{{Name: "rock"}, {Name: "progressive"}},
					Comment: "a comment", Bpm: 127, MediaType: MediaTypeSong, MusicBrainzId: "4321", ChannelCount: 2,
					SamplingRate: 44100, SortName: "sorted title",
					ReplayGain: ReplayGain{TrackGain: 1, AlbumGain: 2, TrackPeak: 3, AlbumPeak: 4, BaseGain: 5, FallbackGain: 6},
				}
				response.Directory.Child = child
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("AlbumWithSongsID3", func() {
		BeforeEach(func() {
			response.AlbumWithSongsID3 = &AlbumWithSongsID3{}
		})
		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				album := AlbumID3{
					Id: "1", Name: "album", Artist: "artist", Genre: "rock",
					Genres:        []ItemGenre{{Name: "rock"}, {Name: "progressive"}},
					MusicBrainzId: "1234", IsCompilation: true, SortName: "sorted album",
					DiscTitles:          DiscTitles{{Disc: 1, Title: "disc 1"}, {Disc: 2, Title: "disc 2"}, {Disc: 3}},
					OriginalReleaseDate: ItemDate{Year: 1994, Month: 2, Day: 4},
					ReleaseDate:         ItemDate{Year: 2000, Month: 5, Day: 10},
				}
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				songs := []Child{{
					Id: "1", IsDir: true, Title: "title", Album: "album", Artist: "artist", Track: 1,
					Year: 1985, Genre: "Rock", CoverArt: "1", Size: 8421341, ContentType: "audio/flac",
					Suffix: "flac", TranscodedContentType: "audio/mpeg", TranscodedSuffix: "mp3",
					Duration: 146, BitRate: 320, Starred: &t, Genres: []ItemGenre{{Name: "rock"}, {Name: "progressive"}},
					Comment: "a comment", Bpm: 127, MediaType: MediaTypeSong, MusicBrainzId: "4321", SortName: "sorted song",
					ReplayGain: ReplayGain{TrackGain: 1, AlbumGain: 2, TrackPeak: 3, AlbumPeak: 4, BaseGain: 5, FallbackGain: 6},
				}}
				response.AlbumWithSongsID3.AlbumID3 = album
				response.AlbumWithSongsID3.Song = songs
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Directory", func() {
		BeforeEach(func() {
			response.Directory = &Directory{Id: "1", Name: "N"}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.Directory.Child = child
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("AlbumList", func() {
		BeforeEach(func() {
			response.AlbumList = &AlbumList{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.AlbumList.Album = child
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("User", func() {
		BeforeEach(func() {
			response.User = &User{Username: "deluan"}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.User.Email = "navidrome@deluan.com"
				response.User.Folder = []int32{1}
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Users", func() {
		BeforeEach(func() {
			u := User{Username: "deluan"}
			response.Users = &Users{User: []User{u}}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				u := User{Username: "deluan"}
				u.Email = "navidrome@deluan.com"
				u.AdminRole = true
				u.Folder = []int32{1}
				response.Users = &Users{User: []User{u}}
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Playlists", func() {
		BeforeEach(func() {
			response.Playlists = &Playlists{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			timestamp, _ := time.Parse(time.RFC3339, "2020-04-11T16:43:00Z04:00")
			BeforeEach(func() {
				pls := make([]Playlist, 2)
				pls[0] = Playlist{
					Id:        "111",
					Name:      "aaa",
					Comment:   "comment",
					SongCount: 2,
					Duration:  120,
					Public:    true,
					Owner:     "admin",
					CoverArt:  "pl-123123123123",
					Created:   timestamp,
					Changed:   timestamp,
				}
				pls[1] = Playlist{Id: "222", Name: "bbb"}
				response.Playlists.Playlist = pls
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Genres", func() {
		BeforeEach(func() {
			response.Genres = &Genres{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				genres := make([]Genre, 3)
				genres[0] = Genre{SongCount: 1000, AlbumCount: 100, Name: "Rock"}
				genres[1] = Genre{SongCount: 500, AlbumCount: 50, Name: "Reggae"}
				genres[2] = Genre{SongCount: 0, AlbumCount: 0, Name: "Pop"}
				response.Genres.Genre = genres
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("AlbumInfo", func() {
		BeforeEach(func() {
			response.AlbumInfo = &AlbumInfo{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.AlbumInfo.SmallImageUrl = "https://lastfm.freetls.fastly.net/i/u/34s/3b54885952161aaea4ce2965b2db1638.png"
				response.AlbumInfo.MediumImageUrl = "https://lastfm.freetls.fastly.net/i/u/64s/3b54885952161aaea4ce2965b2db1638.png"
				response.AlbumInfo.LargeImageUrl = "https://lastfm.freetls.fastly.net/i/u/174s/3b54885952161aaea4ce2965b2db1638.png"
				response.AlbumInfo.LastFmUrl = "https://www.last.fm/music/Cher/Believe"
				response.AlbumInfo.MusicBrainzID = "03c91c40-49a6-44a7-90e7-a700edf97a62"
				response.AlbumInfo.Notes = "Believe is the twenty-third studio album by American singer-actress Cher..."
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("ArtistInfo", func() {
		BeforeEach(func() {
			response.ArtistInfo = &ArtistInfo{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.ArtistInfo.Biography = `Black Sabbath is an English <a target='_blank' href="http://www.last.fm/tag/heavy%20metal" class="bbcode_tag" rel="tag">heavy metal</a> band`
				response.ArtistInfo.MusicBrainzID = "5182c1d9-c7d2-4dad-afa0-ccfeada921a8"
				response.ArtistInfo.LastFmUrl = "https://www.last.fm/music/Black+Sabbath"
				response.ArtistInfo.SmallImageUrl = "https://userserve-ak.last.fm/serve/64/27904353.jpg"
				response.ArtistInfo.MediumImageUrl = "https://userserve-ak.last.fm/serve/126/27904353.jpg"
				response.ArtistInfo.LargeImageUrl = "https://userserve-ak.last.fm/serve/_/27904353/Black+Sabbath+sabbath+1970.jpg"
				response.ArtistInfo.SimilarArtist = []Artist{
					{Id: "22", Name: "Accept"},
					{Id: "101", Name: "Bruce Dickinson"},
					{Id: "26", Name: "Aerosmith"},
				}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("TopSongs", func() {
		BeforeEach(func() {
			response.TopSongs = &TopSongs{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.TopSongs.Song = child
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("SimilarSongs", func() {
		BeforeEach(func() {
			response.SimilarSongs = &SimilarSongs{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.SimilarSongs.Song = child
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("SimilarSongs2", func() {
		BeforeEach(func() {
			response.SimilarSongs2 = &SimilarSongs2{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.SimilarSongs2.Song = child
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("PlayQueue", func() {
		BeforeEach(func() {
			response.PlayQueue = &PlayQueue{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.PlayQueue.Username = "user1"
				response.PlayQueue.Current = "111"
				response.PlayQueue.Position = 243
				response.PlayQueue.Changed = &time.Time{}
				response.PlayQueue.ChangedBy = "a_client"
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.PlayQueue.Entry = child
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Shares", func() {
		BeforeEach(func() {
			response.Shares = &Shares{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				t := time.Time{}
				share := Share{
					ID:          "ABC123",
					Url:         "http://localhost/p/ABC123",
					Description: "Check it out!",
					Username:    "deluan",
					Created:     t,
					Expires:     &t,
					LastVisited: t,
					VisitCount:  2,
				}
				share.Entry = make([]Child, 2)
				share.Entry[0] = Child{Id: "1", Title: "title", Album: "album", Artist: "artist", Duration: 120}
				share.Entry[1] = Child{Id: "2", Title: "title 2", Album: "album", Artist: "artist", Duration: 300}
				response.Shares.Share = []Share{share}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Bookmarks", func() {
		BeforeEach(func() {
			response.Bookmarks = &Bookmarks{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				bmk := Bookmark{
					Position: 123,
					Username: "user2",
					Comment:  "a comment",
					Created:  time.Time{},
					Changed:  time.Time{},
				}
				bmk.Entry = Child{Id: "1", Title: "title", IsDir: false}
				response.Bookmarks.Bookmark = []Bookmark{bmk}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("ScanStatus", func() {
		BeforeEach(func() {
			response.ScanStatus = &ScanStatus{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				timeFmt := "2006-01-02 15:04:00"
				t, _ := time.Parse(timeFmt, timeFmt)
				response.ScanStatus = &ScanStatus{
					Scanning:    true,
					FolderCount: 123,
					Count:       456,
					LastScan:    &t,
				}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("Lyrics", func() {
		BeforeEach(func() {
			response.Lyrics = &Lyrics{}
		})

		Context("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.Lyrics.Artist = "Rick Astley"
				response.Lyrics.Title = "Never Gonna Give You Up"
				response.Lyrics.Value = `Never gonna give you up
				Never gonna let you down
				Never gonna run around and desert you
				Never gonna say goodbye`
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})

		})
	})

	Describe("OpenSubsonicExtensions", func() {
		BeforeEach(func() {
			response.OpenSubsonic = true
			response.OpenSubsonicExtensions = &OpenSubsonicExtensions{}
		})

		Describe("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Describe("with data", func() {
			BeforeEach(func() {
				response.OpenSubsonicExtensions = &OpenSubsonicExtensions{
					OpenSubsonicExtension{Name: "template", Versions: []int32{1, 2}},
				}
			})
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("InternetRadioStations", func() {
		BeforeEach(func() {
			response.InternetRadioStations = &InternetRadioStations{}
		})

		Describe("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Describe("with data", func() {
			BeforeEach(func() {
				radio := make([]Radio, 1)
				radio[0] = Radio{
					ID:          "12345678",
					StreamUrl:   "https://example.com/stream",
					Name:        "Example Stream",
					HomepageUrl: "https://example.com",
				}
				response.InternetRadioStations.Radios = radio
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

	Describe("LyricsList", func() {
		BeforeEach(func() {
			response.LyricsList = &LyricsList{}
		})

		Describe("without data", func() {
			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})

		Describe("with data", func() {
			BeforeEach(func() {
				times := []int64{18800, 22801}
				offset := int64(100)

				response.LyricsList.StructuredLyrics = StructuredLyrics{
					{
						Lang:          "eng",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Offset:        &offset,
						Synced:        true,
						Line: []Line{
							{
								Start: &times[0],
								Value: "We're no strangers to love",
							},
							{
								Start: &times[1],
								Value: "You know the rules and so do I",
							},
						},
					},
					{
						Lang:          "xxx",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Offset:        &offset,
						Synced:        false,
						Line: []Line{
							{
								Value: "We're no strangers to love",
							},
							{
								Value: "You know the rules and so do I",
							},
						},
					},
				}
			})

			It("should match .XML", func() {
				Expect(xml.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
			It("should match .JSON", func() {
				Expect(json.MarshalIndent(response, "", "  ")).To(MatchSnapshot())
			})
		})
	})

})
