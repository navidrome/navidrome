package metadata

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ffmpegExtractor", func() {
	var e *ffmpegExtractor
	BeforeEach(func() {
		e = &ffmpegExtractor{}
	})
	// TODO Need to mock `ffmpeg`
	XContext("Extract", func() {
		It("correctly parses metadata from all files in folder", func() {
			mds, err := e.Extract("tests/fixtures/test.mp3", "tests/fixtures/test.ogg")
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(2))

			m := mds["tests/fixtures/test.mp3"]
			Expect(m.Title()).To(Equal("Song"))
			Expect(m.Album()).To(Equal("Album"))
			Expect(m.Artist()).To(Equal("Artist"))
			Expect(m.AlbumArtist()).To(Equal("Album Artist"))
			Expect(m.Composer()).To(Equal("Composer"))
			Expect(m.Compilation()).To(BeTrue())
			Expect(m.Genre()).To(Equal("Rock"))
			Expect(m.Year()).To(Equal(2014))
			n, t := m.TrackNumber()
			Expect(n).To(Equal(2))
			Expect(t).To(Equal(10))
			n, t = m.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(2))
			Expect(m.HasPicture()).To(BeTrue())
			Expect(m.Duration()).To(Equal(1))
			Expect(m.BitRate()).To(Equal(476))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.mp3"))
			Expect(m.Suffix()).To(Equal("mp3"))
			Expect(m.Size()).To(Equal(60845))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Title()).To(BeEmpty())
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(Equal(3))
			Expect(m.BitRate()).To(Equal(9))
			Expect(m.Suffix()).To(Equal("ogg"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.ogg"))
			Expect(m.Size()).To(Equal(4408))
		})
	})

	Context("extractMetadata", func() {
		It("detects embedded cover art correctly", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Metadata:
    compilation     : 1
  Duration: 00:00:01.02, start: 0.000000, bitrate: 477 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s
    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 600x600 [SAR 1:1 DAR 1:1], 90k tbr, 90k tbn, 90k tbc`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.HasPicture()).To(BeTrue())
		})

		It("detects embedded cover art in ogg containers", func() {
			const output = `
Input #0, ogg, from '/Users/deluan/Music/iTunes/iTunes Media/Music/_Testes/Jamaican In New York/01-02 Jamaican In New York (Album Version).opus':
  Duration: 00:04:28.69, start: 0.007500, bitrate: 139 kb/s
    Stream #0:0(eng): Audio: opus, 48000 Hz, stereo, fltp
    Metadata:
      ALBUM           : Jamaican In New York
      metadata_block_picture: AAAAAwAAAAppbWFnZS9qcGVnAAAAAAAAAAAAAAAAAAAAAAAAAAAAA4Id/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT/2wBDAQMEBAUEBQkFBQkUDQsNFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQ
      TITLE           : Jamaican In New York (Album Version)`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.HasPicture()).To(BeTrue())
		})

		It("gets bitrate from the stream, if available", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 477 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.BitRate()).To(Equal(192))
		})

		It("parses correctly the compilation tag", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Metadata:
    compilation     : 1
  Duration: 00:05:02.63, start: 0.000000, bitrate: 140 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Compilation()).To(BeTrue())
		})

		It("parses duration with milliseconds", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Duration: 00:05:02.63, start: 0.000000, bitrate: 140 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Duration()).To(BeNumerically("~", 302.63, 0.001))
		})

		It("parses stream level tags", func() {
			const output = `
Input #0, ogg, from './01-02 Drive (Teku).opus':
  Metadata:
    ALBUM           : Hot Wheels Acceleracers Soundtrack
  Duration: 00:03:37.37, start: 0.007500, bitrate: 135 kb/s
    Stream #0:0(eng): Audio: opus, 48000 Hz, stereo, fltp
    Metadata:
      TITLE           : Drive (Teku)`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Title()).To(Equal("Drive (Teku)"))
		})

		It("does not overlap top level tags with the stream level tags", func() {
			const output = `
Input #0, mp3, from 'groovin.mp3':
  Metadata:
    title           : Groovin' (feat. Daniel Sneijers, Susanne Alt)
  Duration: 00:03:34.28, start: 0.025056, bitrate: 323 kb/s
    Metadata:
      title           : garbage`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Title()).To(Equal("Groovin' (feat. Daniel Sneijers, Susanne Alt)"))
		})

		It("ignores case in the tag name", func() {
			const output = `
Input #0, flac, from '/Users/deluan/Downloads/06. Back In Black.flac':
  Metadata:
    ALBUM           : Back In Black
    DATE            : 1980.07.25
    disc            : 1
    GENRE           : Hard Rock
    TITLE           : Back In Black
    DISCTOTAL       : 1
    TRACKTOTAL      : 10
    track           : 6
  Duration: 00:04:16.00, start: 0.000000, bitrate: 995 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Title()).To(Equal("Back In Black"))
			Expect(md.Album()).To(Equal("Back In Black"))
			Expect(md.Genre()).To(Equal("Hard Rock"))
			n, t := md.TrackNumber()
			Expect(n).To(Equal(6))
			Expect(t).To(Equal(10))
			n, t = md.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(1))
			Expect(md.Year()).To(Equal(1980))
		})

		// TODO Handle multiline tags
		XIt("parses multiline tags", func() {
			const outputWithMultilineComment = `
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'modulo.m4a':
  Metadata:
    comment         : https://www.mixcloud.com/codigorock/30-minutos-com-saara-saara/
                    :
                    : Tracklist:
                    :
                    : 01. Saara Saara
                    : 02. Carta Corrente
                    : 03. X
                    : 04. Eclipse Lunar
                    : 05. Vírus de Sírius
                    : 06. Doktor Fritz
                    : 07. Wunderbar
                    : 08. Quarta Dimensão
  Duration: 00:26:46.96, start: 0.052971, bitrate: 69 kb/s`
			const expectedComment = `https://www.mixcloud.com/codigorock/30-minutos-com-saara-saara/

Tracklist:

01. Saara Saara
02. Carta Corrente
03. X
04. Eclipse Lunar
05. Vírus de Sírius
06. Doktor Fritz
07. Wunderbar
08. Quarta Dimensão
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", outputWithMultilineComment)
			Expect(md.Comment()).To(Equal(expectedComment))
		})

		It("parses sort tags correctly", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Downloads/椎名林檎 - 加爾基 精液 栗ノ花 - 2003/02 - ドツペルゲンガー.mp3':
  Metadata:
    title-sort      : Dopperugengā
    album           : 加爾基 精液 栗ノ花
    artist          : 椎名林檎
    album_artist    : 椎名林檎
    title           : ドツペルゲンガー
    albumsort       : Kalk Samen Kuri No Hana
    artist_sort     : Shiina, Ringo
    ALBUMARTISTSORT : Shiina, Ringo
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Title()).To(Equal("ドツペルゲンガー"))
			Expect(md.Album()).To(Equal("加爾基 精液 栗ノ花"))
			Expect(md.Artist()).To(Equal("椎名林檎"))
			Expect(md.AlbumArtist()).To(Equal("椎名林檎"))
			Expect(md.SortTitle()).To(Equal("Dopperugengā"))
			Expect(md.SortAlbum()).To(Equal("Kalk Samen Kuri No Hana"))
			Expect(md.SortArtist()).To(Equal("Shiina, Ringo"))
			Expect(md.SortAlbumArtist()).To(Equal("Shiina, Ringo"))
		})

		It("parses tags with spaces in the name", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/Music/Media/_/Wyclef Jean - From the Hut, to the Projects, to the Mansion/10 - The Struggle (interlude).mp3':
  Metadata:
    ALBUM ARTIST    : Wyclef Jean
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.AlbumArtist()).To(Equal("Wyclef Jean"))
		})
	})

	It("creates a valid command line", func() {
		args := e.createProbeCommand([]string{"/music library/one.mp3", "/music library/two.mp3"})
		Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/one.mp3", "-i", "/music library/two.mp3", "-f", "ffmetadata"}))
	})

})
