package ffmpeg

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	var e *Parser
	BeforeEach(func() {
		e = &Parser{}
	})

	Context("extractMetadata", func() {
		It("extracts MusicBrainz custom tags", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio"
		}
	],
	"format": {
		"tags": {
			"ALBUM": "Forever Classics",
			"ARTIST": "Ludwig van Beethoven",
			"TITLE": "Symphony No. 5 in C minor, Op. 67: I. Allegro con brio",
			"MUSICBRAINZ_ALBUMSTATUS": "official",
			"MUSICBRAINZ_ALBUMTYPE": "album",
			"MusicBrainz_AlbumComment": "MP3",
			"Musicbrainz_Albumid": "71eb5e4a-90e2-4a31-a2d1-a96485fcb667",
			"musicbrainz_trackid": "ffe06940-727a-415a-b608-b7e45737f9d8",
			"Musicbrainz_Artistid": "1f9df192-a621-4f54-8850-2c5373b7eac9",
			"Musicbrainz_Albumartistid": "89ad4ac3-39f7-470e-963a-56509c546377",
			"Musicbrainz_Releasegroupid": "708b1ae1-2d3d-34c7-b764-2732b154f5b6",
			"musicbrainz_releasetrackid": "6fee2e35-3049-358f-83be-43b36141028b",
			"CatalogNumber": "PLD 1201"
		}
	}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("catalognumber", []string{"PLD 1201"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_trackid", []string{"ffe06940-727a-415a-b608-b7e45737f9d8"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_albumid", []string{"71eb5e4a-90e2-4a31-a2d1-a96485fcb667"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_artistid", []string{"1f9df192-a621-4f54-8850-2c5373b7eac9"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_albumartistid", []string{"89ad4ac3-39f7-470e-963a-56509c546377"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_albumtype", []string{"album"}))
			Expect(md).To(HaveKeyWithValue("musicbrainz_albumcomment", []string{"MP3"}))
		})

		It("detects embedded cover art correctly", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio"
		},
		{
			"index": 1,
			"codec_type": "video",
			"disposition": {
				"attached_pic": 1
			}
		}
	],
	"format": {}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("has_picture", []string{"true"}))
		})

		It("detects embedded cover art in ogg containers", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio",
			"tags": {
				"ALBUM": "Jamaican In New York",
				"metadata_block_picture": "AAAAAwAAAAppbWFnZS9qcGVnAAAAAAAAAAAAAAAAAAAAAAAAAAAAA4Id/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT/2wBDAQMEBAUEBQkFBQkUDQsNFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQ",
				"TITLE": "Jamaican In New York (Album Version)"
			}
		}
	],
	"format": {}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKey("has_picture"))
		})

		It("gets bitrate from the stream, if available", func() {
			const output2 = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio",
			"bit_rate": "192999"
		}
	],
	"format": {}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output2))
			Expect(md).To(HaveKeyWithValue("bitrate", []string{"192"}))
		})

		It("parses duration with milliseconds", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio"
		}
	],
	"format": {
		"duration": "302.63"
	}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("duration", []string{"302.63"}))
		})

		It("parses stream level tags", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio",
			"tags": {
				"TITLE": "Drive (Teku)"
			}
		}
	]
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("title", []string{"Drive (Teku)"}))
		})

		It("does not overlap top level tags with the stream level tags", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio",
			"tags": {
				"TITLE": "garbage"
			}
		}
	],
	"format": {
		"tags": {
			"title": "Groovin' (feat. Daniel Sneijers, Susanne Alt)"
		}
	}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("title", []string{"Groovin' (feat. Daniel Sneijers, Susanne Alt)", "garbage"}))
		})

		It("parses sort tags correctly", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio"
		}
	],
	"format": {
		"tags": {
			"title-sort": "Dopperugengā",
			"album": "加爾基 精液 栗ノ花",
			"artist": "椎名林檎",
			"album_artist": "椎名林檎",
			"title": "ドツペルゲンガー",
			"albumsort": "Kalk Samen Kuri No Hana",
			"artist_sort": "Shiina, Ringo",
			"ALBUMARTISTSORT": "Shiina, Ringo"
		}
	}
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).To(HaveKeyWithValue("title", []string{"ドツペルゲンガー"}))
			Expect(md).To(HaveKeyWithValue("album", []string{"加爾基 精液 栗ノ花"}))
			Expect(md).To(HaveKeyWithValue("artist", []string{"椎名林檎"}))
			Expect(md).To(HaveKeyWithValue("album_artist", []string{"椎名林檎"}))
			Expect(md).To(HaveKeyWithValue("title-sort", []string{"Dopperugengā"}))
			Expect(md).To(HaveKeyWithValue("albumsort", []string{"Kalk Samen Kuri No Hana"}))
			Expect(md).To(HaveKeyWithValue("artist_sort", []string{"Shiina, Ringo"}))
			Expect(md).To(HaveKeyWithValue("albumartistsort", []string{"Shiina, Ringo"}))
		})

		It("ignores cover comment", func() {
			const output = `
{
	"streams": [
		{
			"index": 0,
			"codec_type": "audio",
			"tags": {
				"comment": "Cover (front)"
			}
		}
	]
}
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", []byte(output))
			Expect(md).ToNot(HaveKey("comment"))
		})
	})

	It("creates a valid command line", func() {
		args := e.createProbeCommand("/music library/one.mp3")
		Expect(args).To(Equal([]string{
			"ffprobe", "-loglevel", "error", "-print_format", "json",
			"-show_format", "-show_streams", "-i", "/music library/one.mp3",
		}))
	})
})
