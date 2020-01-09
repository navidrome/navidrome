//+build linux darwin

// TODO Fix snapshot tests in Windows
// Response Snapshot tests. Only run in Linux and macOS, as they fail in Windows
// Probably because of EOL char differences
package responses_test

import (
	"encoding/json"
	"encoding/xml"
	"time"

	. "github.com/cloudsonic/sonic-server/api/responses"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Responses", func() {
	var response *Subsonic
	BeforeEach(func() {
		response = &Subsonic{Status: "ok", Version: "1.8.0"}
	})

	Describe("EmptyResponse", func() {
		It("should match XML", func() {
			Expect(xml.Marshal(response)).To(MatchSnapshot())
		})
		It("should match JSON", func() {
			Expect(json.Marshal(response)).To(MatchSnapshot())
		})
	})

	Describe("License", func() {
		BeforeEach(func() {
			response.License = &License{Valid: true}
		})
		It("should match XML", func() {
			Expect(xml.Marshal(response)).To(MatchSnapshot())
		})
		It("should match JSON", func() {
			Expect(json.Marshal(response)).To(MatchSnapshot())
		})
	})

	Describe("MusicFolders", func() {
		BeforeEach(func() {
			response.MusicFolders = &MusicFolders{}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				folders := make([]MusicFolder, 2)
				folders[0] = MusicFolder{Id: "111", Name: "aaa"}
				folders[1] = MusicFolder{Id: "222", Name: "bbb"}
				response.MusicFolders.Folders = folders
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("Indexes", func() {
		BeforeEach(func() {
			response.Indexes = &Indexes{LastModified: "1", IgnoredArticles: "A"}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				artists := make([]Artist, 1)
				artists[0] = Artist{Id: "111", Name: "aaa"}
				index := make([]Index, 1)
				index[0] = Index{Name: "A", Artists: artists}
				response.Indexes.Index = index
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("Child", func() {
		Context("with data", func() {
			BeforeEach(func() {
				response.Directory = &Directory{Id: "1", Name: "N"}
				child := make([]Child, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				child[0] = Child{
					Id: "1", IsDir: true, Title: "title", Album: "album", Artist: "artist", Track: 1,
					Year: 1985, Genre: "Rock", CoverArt: "1", Size: "8421341", ContentType: "audio/flac",
					Suffix: "flac", TranscodedContentType: "audio/mpeg", TranscodedSuffix: "mp3",
					Duration: 146, BitRate: 320, Starred: &t,
				}
				response.Directory.Child = child
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("Directory", func() {
		BeforeEach(func() {
			response.Directory = &Directory{Id: "1", Name: "N"}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.Directory.Child = child
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("AlbumList", func() {
		BeforeEach(func() {
			response.AlbumList = &AlbumList{}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.AlbumList.Album = child
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("User", func() {
		BeforeEach(func() {
			response.User = &User{Username: "deluan"}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				response.User.Email = "cloudsonic@deluan.com"
				response.User.Folder = []int{1}
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})

	Describe("Playlists", func() {
		BeforeEach(func() {
			response.Playlists = &Playlists{}
		})

		Context("without data", func() {
			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})

		Context("with data", func() {
			BeforeEach(func() {
				pls := make([]Playlist, 2)
				pls[0] = Playlist{Id: "111", Name: "aaa"}
				pls[1] = Playlist{Id: "222", Name: "bbb"}
				response.Playlists.Playlist = pls
			})

			It("should match XML", func() {
				Expect(xml.Marshal(response)).To(MatchSnapshot())
			})
			It("should match JSON", func() {
				Expect(json.Marshal(response)).To(MatchSnapshot())
			})
		})
	})
})
