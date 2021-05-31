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
			Expect(m.Duration()).To(BeNumerically("~", 1.03, 0.001))
			Expect(m.BitRate()).To(Equal(192))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.mp3"))
			Expect(m.Suffix()).To(Equal("mp3"))
			Expect(m.Size()).To(Equal(int64(51876)))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Title()).To(BeEmpty())
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(BeNumerically("~", 1.04, 0.001))
			Expect(m.BitRate()).To(Equal(16))
			Expect(m.Suffix()).To(Equal("ogg"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.ogg"))
			Expect(m.Size()).To(Equal(int64(5065)))
		})
	})

	Context("extractMetadata", func() {
		It("extracts MusicBrainz custom tags", func() {
			const output = `
Input #0, ape, from './Capture/02 01 - Symphony No. 5 in C minor, Op. 67 I. Allegro con brio - Ludwig van Beethoven.ape':
  Metadata:
    ALBUM           : Forever Classics
    ARTIST          : Ludwig van Beethoven
    TITLE           : Symphony No. 5 in C minor, Op. 67: I. Allegro con brio
    MUSICBRAINZ_ALBUMSTATUS: official
    MUSICBRAINZ_ALBUMTYPE: album
    MusicBrainz_AlbumComment: MP3
    Musicbrainz_Albumid: 71eb5e4a-90e2-4a31-a2d1-a96485fcb667
    musicbrainz_trackid: ffe06940-727a-415a-b608-b7e45737f9d8
    Musicbrainz_Artistid: 1f9df192-a621-4f54-8850-2c5373b7eac9
    Musicbrainz_Albumartistid: 89ad4ac3-39f7-470e-963a-56509c546377
    Musicbrainz_Releasegroupid: 708b1ae1-2d3d-34c7-b764-2732b154f5b6
    musicbrainz_releasetrackid: 6fee2e35-3049-358f-83be-43b36141028b
    CatalogNumber   : PLD 1201
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.CatalogNum()).To(Equal("PLD 1201"))
			Expect(md.MbzTrackID()).To(Equal("ffe06940-727a-415a-b608-b7e45737f9d8"))
			Expect(md.MbzAlbumID()).To(Equal("71eb5e4a-90e2-4a31-a2d1-a96485fcb667"))
			Expect(md.MbzArtistID()).To(Equal("1f9df192-a621-4f54-8850-2c5373b7eac9"))
			Expect(md.MbzAlbumArtistID()).To(Equal("89ad4ac3-39f7-470e-963a-56509c546377"))
			Expect(md.MbzAlbumType()).To(Equal("album"))
			Expect(md.MbzAlbumComment()).To(Equal("MP3"))
		})

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

		It("detects embedded cover art in ffmpeg 4.4 output", func() {
			const output = `

Input #0, flac, from '/run/media/naomi/Archivio/Musica/Katy Perry/Chained to the Rhythm/01 Katy Perry featuring Skip Marley - Chained to the Rhythm.flac':
  Metadata:
    ARTIST          : Katy Perry featuring Skip Marley
  Duration: 00:03:57.91, start: 0.000000, bitrate: 983 kb/s
  Stream #0:0: Audio: flac, 44100 Hz, stereo, s16
  Stream #0:1: Video: mjpeg (Baseline), yuvj444p(pc, bt470bg/unknown/unknown), 599x518, 90k tbr, 90k tbn, 90k tbc (attached pic)
    Metadata:
      comment         : Cover (front)`
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

		It("parses multiline tags", func() {
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
08. Quarta Dimensão`
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

		It("ignores cover comment", func() {
			const output = `
Input #0, mp3, from './Edie Brickell/Picture Perfect Morning/01-01 Tomorrow Comes.mp3':
  Metadata:
    title           : Tomorrow Comes
    artist          : Edie Brickell
  Duration: 00:03:56.12, start: 0.000000, bitrate: 332 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, s16p, 320 kb/s
    Stream #0:1: Video: mjpeg, yuvj420p(pc, bt470bg/unknown/unknown), 1200x1200 [SAR 72:72 DAR 1:1], 90k tbr, 90k tbn, 90k tbc
    Metadata:
      comment         : Cover (front)`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md.Comment()).To(Equal(""))
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

	It("parses an integer TBPM tag", func() {
		const output = `
		Input #0, mp3, from 'tests/fixtures/test.mp3':
		  Metadata:
		    TBPM            : 123`
		md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
		Expect(md.Bpm()).To(Equal(123))
	})

	It("parses and rounds a floating point fBPM tag", func() {
		const output = `
		Input #0, ogg, from 'tests/fixtures/test.ogg':
  		  Metadata:
	        FBPM            : 141.7`
		md, _ := e.extractMetadata("tests/fixtures/test.ogg", output)
		Expect(md.Bpm()).To(Equal(142))
	})
})
