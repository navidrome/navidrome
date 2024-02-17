package ffmpeg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extractor", func() {
	var e *Extractor
	BeforeEach(func() {
		e = &Extractor{}
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
			Expect(md).To(SatisfyAll(
				HaveKeyWithValue("catalognumber", []string{"PLD 1201"}),
				HaveKeyWithValue("musicbrainz_trackid", []string{"ffe06940-727a-415a-b608-b7e45737f9d8"}),
				HaveKeyWithValue("musicbrainz_albumid", []string{"71eb5e4a-90e2-4a31-a2d1-a96485fcb667"}),
				HaveKeyWithValue("musicbrainz_artistid", []string{"1f9df192-a621-4f54-8850-2c5373b7eac9"}),
				HaveKeyWithValue("musicbrainz_albumartistid", []string{"89ad4ac3-39f7-470e-963a-56509c546377"}),
				HaveKeyWithValue("musicbrainz_albumtype", []string{"album"}),
				HaveKeyWithValue("musicbrainz_albumcomment", []string{"MP3"}),
			))
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
			Expect(md).To(HaveKeyWithValue("has_picture", []string{"true"}))
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
			Expect(md).To(HaveKeyWithValue("has_picture", []string{"true"}))
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
			Expect(md).To(HaveKey("metadata_block_picture"))
			md = md.Map(e.CustomMappings())
			Expect(md).To(HaveKey("has_picture"))
		})

		It("detects embedded cover art in m4a containers", func() {
			const output = `
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'Putumayo Presents_ Euro Groove/01 Destins et Désirs.m4a':
  Metadata:
    album           : Putumayo Presents: Euro Groove
  Duration: 00:05:15.81, start: 0.047889, bitrate: 133 kb/s
  Stream #0:0[0x1](und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 125 kb/s (default)
    Metadata:
      creation_time   : 2008-03-11T21:03:23.000000Z
      vendor_id       : [0][0][0][0]
  Stream #0:1[0x0]: Video: png, rgb24(pc), 350x350, 90k tbr, 90k tbn (attached pic)
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("has_picture", []string{"true"}))
		})

		It("gets bitrate from the stream, if available", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 477 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("bitrate", []string{"192"}))
		})

		It("parses duration with milliseconds", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Duration: 00:05:02.63, start: 0.000000, bitrate: 140 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("duration", []string{"302.63"}))
		})

		It("parse flac bitrates", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 477 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("channels", []string{"2"}))
		})

		It("parse channels from the stream with bitrate", func() {
			const output = `
Input #0, flac, from '/Users/deluan/Music/Music/Media/__/Crazy For You/01-01 Crazy For You.flac':
  Metadata:
    TITLE           : Crazy For You
  Duration: 00:04:13.00, start: 0.000000, bitrate: 852 kb/s
  Stream #0:0: Audio: flac, 44100 Hz, stereo, s16
  Stream #0:1: Video: mjpeg (Progressive), yuvj444p(pc, bt470bg/unknown/unknown), 600x600, 90k tbr, 90k tbn, 90k tbc (attached pic)
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("bitrate", []string{"852"}))
		})

		It("parse 7.1 channels from the stream", func() {
			const output = `
Input #0, wav, from '/Users/deluan/Music/Music/Media/_/multichannel/Nums_7dot1_24_48000.wav':
  Duration: 00:00:09.05, bitrate: 9216 kb/s
  Stream #0:0: Audio: pcm_s24le ([1][0][0][0] / 0x0001), 48000 Hz, 7.1, s32 (24 bit), 9216 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("channels", []string{"8"}))
		})

		It("parse channels from the stream without bitrate", func() {
			const output = `
Input #0, flac, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.flac':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 1371 kb/s
    Stream #0:0: Audio: flac, 44100 Hz, stereo, fltp, s32 (24 bit)`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("channels", []string{"2"}))
		})

		It("parse channels from the stream with lang", func() {
			const output = `
Input #0, flac, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.m4a':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 1371 kb/s
    Stream #0:0(eng): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 262 kb/s (default)`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("channels", []string{"2"}))
		})

		It("parse channels from the stream with lang 2", func() {
			const output = `
Input #0, flac, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.m4a':
  Duration: 00:00:01.02, start: 0.000000, bitrate: 1371 kb/s
    Stream #0:0(eng): Audio: vorbis, 44100 Hz, stereo, fltp, 192 kb/s`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("channels", []string{"2"}))
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
			Expect(md).To(HaveKeyWithValue("title", []string{"Drive (Teku)"}))
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
			Expect(md).To(HaveKeyWithValue("title", []string{"Groovin' (feat. Daniel Sneijers, Susanne Alt)", "garbage"}))
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
			Expect(md).To(HaveKeyWithValue("comment", []string{expectedComment}))
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
			Expect(md).To(SatisfyAll(
				HaveKeyWithValue("title", []string{"ドツペルゲンガー"}),
				HaveKeyWithValue("album", []string{"加爾基 精液 栗ノ花"}),
				HaveKeyWithValue("artist", []string{"椎名林檎"}),
				HaveKeyWithValue("album_artist", []string{"椎名林檎"}),
				HaveKeyWithValue("title-sort", []string{"Dopperugengā"}),
				HaveKeyWithValue("albumsort", []string{"Kalk Samen Kuri No Hana"}),
				HaveKeyWithValue("artist_sort", []string{"Shiina, Ringo"}),
				HaveKeyWithValue("albumartistsort", []string{"Shiina, Ringo"}),
			))
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
			Expect(md).ToNot(HaveKey("comment"))
		})

		It("parses tags with spaces in the name", func() {
			const output = `
Input #0, mp3, from '/Users/deluan/Music/Music/Media/_/Wyclef Jean - From the Hut, to the Projects, to the Mansion/10 - The Struggle (interlude).mp3':
  Metadata:
    ALBUM ARTIST    : Wyclef Jean
`
			md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
			Expect(md).To(HaveKeyWithValue("album artist", []string{"Wyclef Jean"}))
		})
	})

	It("parses an integer TBPM tag", func() {
		const output = `
		Input #0, mp3, from 'tests/fixtures/test.mp3':
		  Metadata:
		    TBPM            : 123`
		md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
		Expect(md).To(HaveKeyWithValue("tbpm", []string{"123"}))
	})

	It("parses and rounds a floating point fBPM tag", func() {
		const output = `
		Input #0, ogg, from 'tests/fixtures/test.ogg':
  		  Metadata:
	        FBPM            : 141.7`
		md, _ := e.extractMetadata("tests/fixtures/test.ogg", output)
		Expect(md).To(HaveKeyWithValue("fbpm", []string{"141.7"}))
	})

	It("parses replaygain data correctly", func() {
		const output = `
		Input #0, mp3, from 'test.mp3':
			Metadata:
				REPLAYGAIN_ALBUM_PEAK: 0.9125
				REPLAYGAIN_TRACK_PEAK: 0.4512
				REPLAYGAIN_TRACK_GAIN: -1.48 dB
				REPLAYGAIN_ALBUM_GAIN: +3.21518 dB
				Side data:
					replaygain: track gain - -1.480000, track peak - 0.000011, album gain - 3.215180, album peak - 0.000021, 
		`
		md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
		Expect(md).To(SatisfyAll(
			HaveKeyWithValue("replaygain_track_gain", []string{"-1.48 dB"}),
			HaveKeyWithValue("replaygain_track_peak", []string{"0.4512"}),
			HaveKeyWithValue("replaygain_album_gain", []string{"+3.21518 dB"}),
			HaveKeyWithValue("replaygain_album_peak", []string{"0.9125"}),
		))
	})

	It("parses lyrics with language code", func() {
		const output = `
		Input #0, mp3, from 'test.mp3':
  		Metadata:
				lyrics-eng      : [00:00.00]This is
												: [00:02.50]English
				lyrics-xxx      : [00:00.00]This is
												: [00:02.50]unspecified
		`
		md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
		Expect(md).To(SatisfyAll(
			HaveKeyWithValue("lyrics-eng", []string{
				"[00:00.00]This is\n[00:02.50]English",
			}),
			HaveKeyWithValue("lyrics-xxx", []string{
				"[00:00.00]This is\n[00:02.50]unspecified",
			}),
		))
	})

	It("parses normal LYRICS tag", func() {
		const output = `
		Input #0, mp3, from 'test.mp3':
  		Metadata:
				LYRICS      : [00:00.00]This is
										: [00:02.50]English
		`
		md, _ := e.extractMetadata("tests/fixtures/test.mp3", output)
		Expect(md).To(HaveKeyWithValue("lyrics", []string{
			"[00:00.00]This is\n[00:02.50]English",
		}))
	})
})
