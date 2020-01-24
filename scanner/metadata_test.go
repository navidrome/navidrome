package scanner

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	It("correctly parses metadata from all files in folder", func() {
		mds, err := ExtractAllMetadata("../tests/fixtures")
		Expect(err).NotTo(HaveOccurred())
		Expect(mds).To(HaveLen(3))

		m := mds["../tests/fixtures/test.mp3"]
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
		Expect(m.FilePath()).To(Equal("../tests/fixtures/test.mp3"))
		Expect(m.Suffix()).To(Equal("mp3"))
		Expect(m.Size()).To(Equal(60845))

		m = mds["../tests/fixtures/test.ogg"]
		Expect(err).To(BeNil())
		Expect(m.Title()).To(BeEmpty())
		Expect(m.HasPicture()).To(BeFalse())
		Expect(m.Duration()).To(Equal(3))
		Expect(m.BitRate()).To(Equal(9))
		Expect(m.Suffix()).To(Equal("ogg"))
		Expect(m.FilePath()).To(Equal("../tests/fixtures/test.ogg"))
		Expect(m.Size()).To(Equal(4408))
	})

	It("returns error if path does not exist", func() {
		_, err := ExtractAllMetadata("./INVALID/PATH")
		Expect(err).To(HaveOccurred())
	})

	It("returns empty map if there are no audio files in path", func() {
		Expect(ExtractAllMetadata(".")).To(BeEmpty())
	})

	Context("extractMetadata", func() {
		It("parses correctly the compilation tag", func() {
			const outputWithOverlappingTitleTag = `
Input #0, mp3, from '/Users/deluan/Music/iTunes/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/09 Pablo's Blues.mp3':
  Metadata:
    iTunSMPB        :  00000000 000002D6 00000216 0000000000CB9F94 02000003 0049D539 00000000 00000000 00000000 00000000 00000000 00000000
    iTunNORM        :  000002FF 0000027E 00000FEF 00000C17 0002E647 00044605 00007F02 00007A92 0000273E 0000273E
    title           : Pablo's Blues
    artist          : Gare Du Nord
    album           : Putumayo Presents Blues Lounge
    TT1             : Putumayo
    track           : 9/10
    compilation     : 1
    genre           : Blues
    date            : 2004
  Duration: 00:05:02.63, start: 0.000000, bitrate: 140 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 128 kb/s
    Stream #0:1: Video: png, rgb24(pc), 500x478 [SAR 2835:2835 DAR 250:239], 90k tbr, 90k tbn, 90k tbc
    Metadata:
      comment         : Other`
			md, _ := extractMetadata("pablo's blues.mp3", outputWithOverlappingTitleTag)
			Expect(md.Compilation()).To(BeTrue())
		})

		It("parses correct the title without overlapping with the stream tag", func() {
			const outputWithOverlappingTitleTag = `
Input #0, mp3, from 'groovin.mp3':
  Metadata:
    title           : Groovin' (feat. Daniel Sneijers, Susanne Alt)
    artist          : Bone 40
    track           : 1
    album           : Groovin'
    album_artist    : Bone 40
    comment         : Visit http://bone40.bandcamp.com
    date            : 2016
  Duration: 00:03:34.28, start: 0.025056, bitrate: 323 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 320 kb/s
    Metadata:
      encoder         : LAME3.99r
    Side data:
      replaygain: track gain - -6.000000, track peak - unknown, album gain - unknown, album peak - unknown,
    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 700x700 [SAR 72:72 DAR 1:1], 90k tbr, 90k tbn, 90k tbc
    Metadata:
      title           : cover
      comment         : Cover (front)
At least one output file must be specified`
			md, _ := extractMetadata("groovin.mp3", outputWithOverlappingTitleTag)
			Expect(md.Title()).To(Equal("Groovin' (feat. Daniel Sneijers, Susanne Alt)"))
		})

		It("ignores case in the tag name", func() {
			const outputWithOverlappingTitleTag = `
Input #0, flac, from '/Users/deluan/Downloads/06. Back In Black.flac':
  Metadata:
    ALBUM           : Back In Black
    album_artist    : AC/DC
    ARTIST          : AC/DC
    COMPOSER        : Angus Young;Malcolm Young;Brian Johnson
    DATE            : 1980.07.25
    disc            : 1
    GENRE           : Hard Rock
    LANGUAGE        : EN
    RATING          : 2
    TITLE           : Back In Black
    DISCTOTAL       : 1
    TRACKTOTAL      : 10
    track           : 6
    REPLAYGAIN_TRACK_GAIN: -8.51 dB
    REPLAYGAIN_TRACK_PEAK: 0.998322
  Duration: 00:04:16.00, start: 0.000000, bitrate: 995 kb/s
    Stream #0:0: Audio: flac, 44100 Hz, stereo, s16
    Side data:
      replaygain: track gain - -8.510000, track peak - 0.000023, album gain - unknown, album peak - unknown,`
			md, _ := extractMetadata("groovin.mp3", outputWithOverlappingTitleTag)
			Expect(md.Title()).To(Equal("Back In Black"))
			Expect(md.Album()).To(Equal("Back In Black"))
			Expect(md.Genre()).To(Equal("Hard Rock"))
			n, t := md.TrackNumber()
			Expect(n).To(Equal(6))
			Expect(t).To(Equal(10))
			n, t = md.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(1))

		})

		// TODO Handle multiline tags
		XIt("parses multiline tags", func() {
			const outputWithMultilineComment = `
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'modulo.m4a':
  Metadata:
    major_brand     : mp42
    minor_version   : 0
    compatible_brands: M4A mp42isom
    creation_time   : 2014-05-10T21:11:57.000000Z
    iTunSMPB        :  00000000 00000920 000000E0 00000000021CA200 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000
    encoder         : Nero AAC codec / 1.5.4.0
    title           : Módulo Especial
    artist          : Saara Saara
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
    album           : Módulo Especial
    genre           : Electronic
    track           : 1
  Duration: 00:26:46.96, start: 0.052971, bitrate: 69 kb/s
    Chapter #0:0: start 0.105941, end 1607.013149
    Metadata:
      title           :
    Stream #0:0(und): Audio: aac (HE-AAC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 69 kb/s (default)
    Metadata:
      creation_time   : 2014-05-10T21:11:57.000000Z
      handler_name    : Sound Media Handler
At least one output file must be specified`
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
			md, _ := extractMetadata("modulo.mp3", outputWithMultilineComment)
			Expect(md.Comment()).To(Equal(expectedComment))
		})
	})
})
