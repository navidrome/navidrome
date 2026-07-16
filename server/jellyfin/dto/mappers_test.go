package dto

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mappers", func() {
	It("maps a song to an Audio BaseItemDto", func() {
		mf := model.MediaFile{
			ID: "song-1", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", TrackNumber: 3, DiscNumber: 1,
			Year: 1999, Duration: 60, Size: 2_500_000,
		}
		mf.PlayCount = 2
		mf.Starred = true
		item := SongToBaseItem(mf, nil)
		Expect(item.Type).To(Equal("Audio"))
		Expect(item.MediaType).To(Equal("Audio"))
		Expect(item.IsFolder).To(BeFalse())
		Expect(item.LocationType).To(Equal("FileSystem"))
		Expect(item.Id).To(Equal(EncodeID("song-1")))
		Expect(item.AlbumId).To(Equal(EncodeID("alb-1")))
		Expect(item.ParentId).To(Equal(EncodeID("alb-1")))
		Expect(item.RunTimeTicks).To(Equal(int64(600_000_000)))
		Expect(*item.IndexNumber).To(Equal(3))
		Expect(item.UserData.IsFavorite).To(BeTrue())
		Expect(item.UserData.PlayCount).To(Equal(2))
		Expect(item.UserData.Played).To(BeTrue())
		Expect(item.UserData.Key).To(Equal(EncodeID("song-1")))
		Expect(item.UserData.ItemId).To(Equal(EncodeID("song-1")))
		Expect(item.ImageBlurHashes["Primary"]).To(HaveKey(item.AlbumPrimaryImageTag))
		Expect(item.ImageBlurHashes["Primary"][item.AlbumPrimaryImageTag]).To(HaveLen(6))
	})

	Describe("Fields gating (matches real Jellyfin)", func() {
		mf := model.MediaFile{ID: "s1", Title: "Song", Size: 2_500_000, Suffix: "mp3", Duration: 60,
			SortTitle: "sort song", Lyrics: `[{"line":[{"value":"la"}]}]`}

		It("omits MediaSources and SortName when Fields does not ask for them", func() {
			item := SongToBaseItem(mf, nil)
			Expect(item.MediaSources).To(BeNil())
			Expect(item.SortName).To(BeEmpty())
		})

		It("includes MediaSources only when Fields=MediaSources", func() {
			item := SongToBaseItem(mf, ParseFields("ChildCount,MediaSources,SortName"))
			Expect(item.MediaSources).To(HaveLen(1))
			Expect(item.MediaSources[0].Size).To(Equal(int64(2_500_000)))
		})

		It("includes SortName (from the sort title) only when Fields=SortName", func() {
			Expect(SongToBaseItem(mf, ParseFields("SortName")).SortName).To(Equal("sort song"))
		})

		It("sets HasLyrics from the media file's lyrics", func() {
			Expect(SongToBaseItem(mf, nil).HasLyrics).To(BeTrue())
			Expect(SongToBaseItem(model.MediaFile{ID: "s2", Title: "No Lyrics"}, nil).HasLyrics).To(BeFalse())
			// "[]" is the no-lyrics sentinel, not a truthy value.
			Expect(SongToBaseItem(model.MediaFile{ID: "s3", Title: "Empty Lyrics", Lyrics: "[]"}, nil).HasLyrics).To(BeFalse())
		})
	})

	It("omits ImageBlurHashes when a song has no album", func() {
		mf := model.MediaFile{ID: "song-noalbum", Title: "Song", Duration: 60}
		item := SongToBaseItem(mf, nil)
		Expect(item.AlbumPrimaryImageTag).To(BeEmpty())
		Expect(item.ImageBlurHashes).To(BeNil())
	})

	It("sets DateCreated from the media file's CreatedAt", func() {
		mf := model.MediaFile{ID: "s1", Title: "Song", CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)}
		Expect(SongToBaseItem(mf, nil).DateCreated).To(Equal("2024-01-15T10:30:00Z"))
	})

	It("omits DateCreated when CreatedAt is the zero time", func() {
		Expect(SongToBaseItem(model.MediaFile{ID: "s1", Title: "Song"}, nil).DateCreated).To(BeEmpty())
	})

	It("sets ArtistItems and AlbumArtists (encoded ids) from the track and album artist", func() {
		mf := model.MediaFile{
			ID: "s1", Title: "Song",
			Artist: "The Band", ArtistID: "ar-1",
			AlbumArtist: "Various", AlbumArtistID: "ar-2",
		}
		item := SongToBaseItem(mf, nil)
		Expect(item.ArtistItems).To(Equal([]NameGuidPair{{Name: "The Band", Id: EncodeID("ar-1")}}))
		Expect(item.AlbumArtists).To(Equal([]NameGuidPair{{Name: "Various", Id: EncodeID("ar-2")}}))
	})

	It("omits ArtistItems when the track has no artist id", func() {
		Expect(SongToBaseItem(model.MediaFile{ID: "s1", Title: "Song", Artist: "X"}, nil).ArtistItems).To(BeNil())
	})

	It("builds a MediaSourceInfo from a media file", func() {
		mf := model.MediaFile{ID: "s1", Size: 5242880, Suffix: "mp3", BitRate: 320, Duration: 100}
		src := MediaSourceFromMediaFile(mf)
		Expect(src.Id).To(Equal(EncodeID("s1")))
		Expect(src.Size).To(Equal(int64(5242880)))
		Expect(src.Container).To(Equal("mp3"))
		Expect(src.Bitrate).To(Equal(320_000))
		Expect(src.RunTimeTicks).To(Equal(int64(1_000_000_000)))
		Expect(src.Protocol).To(Equal("Http"))
		Expect(src.SupportsDirectPlay).To(BeTrue())
	})

	It("populates MediaStreams with a single Audio stream so Finamp can size downloads", func() {
		mf := model.MediaFile{
			ID: "s1", Size: 5242880, Suffix: "mp3", BitRate: 320, Duration: 100,
			Channels: 2, SampleRate: 44100, Codec: "mp3",
		}
		src := MediaSourceFromMediaFile(mf)
		Expect(src.MediaStreams).To(HaveLen(1))
		stream := src.MediaStreams[0]
		Expect(stream.Type).To(Equal("Audio"))
		Expect(stream.Channels).To(Equal(2))
		Expect(stream.SampleRate).To(Equal(44100))
		Expect(stream.BitRate).To(Equal(320_000))
		Expect(stream.Codec).To(Equal("mp3"))
		Expect(stream.ChannelLayout).To(Equal("stereo"))
	})

	It("serializes all Finamp-required MediaSourceInfo bools and arrays, never as null", func() {
		mf := model.MediaFile{ID: "s1", Size: 5242880, Suffix: "mp3", BitRate: 320, Duration: 100}
		src := MediaSourceFromMediaFile(mf)
		b, err := json.Marshal(src)
		Expect(err).ToNot(HaveOccurred())
		j := string(b)
		Expect(j).To(ContainSubstring(`"SupportsProbing":true`))
		Expect(j).To(ContainSubstring(`"IsInfiniteStream":false`))
		Expect(j).To(ContainSubstring(`"RequiresOpening":false`))
		Expect(j).To(ContainSubstring(`"MediaAttachments":[]`))
		Expect(j).To(ContainSubstring(`"Formats":[]`))
	})

	It("serializes MediaStream's required non-nullable bools, never omitted", func() {
		stream := MediaStream{Type: "Audio", Index: 0}
		b, err := json.Marshal(stream)
		Expect(err).ToNot(HaveOccurred())
		j := string(b)
		Expect(j).To(ContainSubstring(`"Type":"Audio"`))
		Expect(j).To(ContainSubstring(`"IsDefault":false`))
		Expect(j).To(ContainSubstring(`"IsInterlaced":false`))
		Expect(j).To(ContainSubstring(`"IsForced":false`))
		Expect(j).To(ContainSubstring(`"IsExternal":false`))
		Expect(j).To(ContainSubstring(`"IsTextSubtitleStream":false`))
		Expect(j).To(ContainSubstring(`"SupportsExternalStream":false`))
	})

	Describe("Lyric media stream advertising", func() {
		It("adds a Lyric media stream when the file has embedded lyrics", func() {
			mf := model.MediaFile{ID: "s1", Lyrics: `[{"line":[{"value":"la"}]}]`}
			src := MediaSourceFromMediaFile(mf)
			Expect(src.MediaStreams).To(HaveLen(2))
			Expect(src.MediaStreams[0].Type).To(Equal("Audio"))
			Expect(src.MediaStreams[1].Type).To(Equal("Lyric"))
			Expect(src.MediaStreams[1].Index).To(Equal(1))
			Expect(src.MediaStreams[1].IsExternal).To(BeTrue())
		})

		It("emits only the Audio stream without lyrics", func() {
			src := MediaSourceFromMediaFile(model.MediaFile{ID: "s1"})
			Expect(src.MediaStreams).To(HaveLen(1))
			Expect(src.MediaStreams[0].Type).To(Equal("Audio"))
		})

		It("emits only the Audio stream for the post-scan empty-lyrics sentinel", func() {
			src := MediaSourceFromMediaFile(model.MediaFile{ID: "s1", Lyrics: "[]"})
			Expect(src.MediaStreams).To(HaveLen(1))
		})
	})

	It("omits IndexNumber and ParentIndexNumber when track/disc numbers are untagged", func() {
		mf := model.MediaFile{
			ID: "song-2", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", TrackNumber: 0, DiscNumber: 0,
			Duration: 60,
		}
		item := SongToBaseItem(mf, nil)
		Expect(item.IndexNumber).To(BeNil())
		Expect(item.ParentIndexNumber).To(BeNil())
	})

	It("maps PlayDate to UserData.LastPlayedDate", func() {
		playDate := time.Date(2023, 5, 17, 12, 30, 0, 0, time.UTC)
		mf := model.MediaFile{
			ID: "song-3", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", Duration: 60,
		}
		mf.PlayDate = &playDate
		item := SongToBaseItem(mf, nil)
		Expect(item.UserData.LastPlayedDate).NotTo(BeNil())
		Expect(*item.UserData.LastPlayedDate).To(Equal(playDate.Format(time.RFC3339)))
	})

	It("maps an album to a MusicAlbum folder item", func() {
		al := model.Album{ID: "alb-1", Name: "Alb", AlbumArtist: "AA", AlbumArtistID: "art-1", MaxYear: 1999, SongCount: 10}
		item := AlbumToBaseItem(al)
		Expect(item.Type).To(Equal("MusicAlbum"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal(EncodeID("alb-1")))
		Expect(item.ParentId).To(Equal(EncodeID("art-1")))
		Expect(item.AlbumArtists).To(HaveLen(1))
		Expect(item.AlbumArtists[0].Id).To(Equal(EncodeID("art-1")))
		Expect(item.ArtistItems).To(Equal(item.AlbumArtists))
		Expect(*item.ProductionYear).To(Equal(1999))
		Expect(*item.ChildCount).To(Equal(10))
		Expect(item.ImageBlurHashes["Primary"]).To(HaveKey(item.ImageTags["Primary"]))
		Expect(item.ImageBlurHashes["Primary"][item.ImageTags["Primary"]]).To(HaveLen(6))
	})

	It("maps an artist to a MusicArtist folder item", func() {
		ar := model.Artist{ID: "art-1", Name: "AA", AlbumCount: 2, SongCount: 20}
		item := ArtistToBaseItem(ar)
		Expect(item.Type).To(Equal("MusicArtist"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal(EncodeID("art-1")))
		Expect(*item.AlbumCount).To(Equal(2))
	})

	It("maps a genre to a MusicGenre folder item", func() {
		g := model.Genre{ID: "genre-1", Name: "Rock"}
		item := GenreToBaseItem(g)
		Expect(item.Type).To(Equal("MusicGenre"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal(EncodeID("genre-1")))
		Expect(item.Name).To(Equal("Rock"))
	})

	Describe("premiereDate", func() {
		// Finamp re-sorts "Latest Releases" client-side by PremiereDate; absent values sort arbitrarily.
		It("serializes a full date", func() {
			mf := model.MediaFile{ID: "s1", Title: "Song", Date: "2007-02-01", Year: 2007}
			item := SongToBaseItem(mf, nil)
			Expect(*item.PremiereDate).To(Equal("2007-02-01T00:00:00Z"))
		})

		It("pads a year-only date so clients can parse it", func() {
			mf := model.MediaFile{ID: "s1", Title: "Song", Date: "2007", Year: 2007}
			Expect(*SongToBaseItem(mf, nil).PremiereDate).To(Equal("2007-01-01T00:00:00Z"))
		})

		It("pads a year-month date", func() {
			mf := model.MediaFile{ID: "s1", Title: "Song", Date: "2007-02"}
			Expect(*SongToBaseItem(mf, nil).PremiereDate).To(Equal("2007-02-01T00:00:00Z"))
		})

		It("falls back to the year when no date tag exists", func() {
			mf := model.MediaFile{ID: "s1", Title: "Song", Year: 1999}
			Expect(*SongToBaseItem(mf, nil).PremiereDate).To(Equal("1999-01-01T00:00:00Z"))
		})

		It("is omitted when the track has no date at all", func() {
			Expect(SongToBaseItem(model.MediaFile{ID: "s1", Title: "Song"}, nil).PremiereDate).To(BeNil())
		})

		It("is set on albums from their date, falling back to MaxYear", func() {
			Expect(*AlbumToBaseItem(model.Album{ID: "a1", Date: "2013-09-06"}).PremiereDate).To(Equal("2013-09-06T00:00:00Z"))
			Expect(*AlbumToBaseItem(model.Album{ID: "a2", MaxYear: 2013}).PremiereDate).To(Equal("2013-01-01T00:00:00Z"))
			Expect(AlbumToBaseItem(model.Album{ID: "a3"}).PremiereDate).To(BeNil())
		})
	})

	It("maps a playlist to a Playlist BaseItemDto", func() {
		p := model.Playlist{
			ID: "pl-1", Name: "Chill", SongCount: 7, Duration: 120,
			Annotations: model.Annotations{Starred: true, Rating: 4, PlayCount: 2},
		}
		item := PlaylistToBaseItem(p)
		Expect(item.Type).To(Equal("Playlist"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal(EncodeID("pl-1")))
		Expect(item.Name).To(Equal("Chill"))
		Expect(item.MediaType).To(Equal("Audio"))
		Expect(*item.ChildCount).To(Equal(7))
		Expect(item.RunTimeTicks).To(Equal(int64(1_200_000_000)))
		Expect(item.UserData.IsFavorite).To(BeTrue())
		Expect(item.UserData.PlayCount).To(Equal(2))
		Expect(*item.UserData.Rating).To(Equal(8.0))
		tag := item.ImageTags["Primary"]
		Expect(tag).ToNot(BeEmpty())
		Expect(item.ImageBlurHashes["Primary"]).To(HaveKey(tag))
		Expect(item.ImageBlurHashes["Primary"][tag]).To(HaveLen(6))
	})

	It("changes the playlist image tag and blurhash when the playlist is updated (cover upload)", func() {
		p := model.Playlist{ID: "pl-1", Name: "Chill", UpdatedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
		before := PlaylistToBaseItem(p)
		p.UpdatedAt = time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
		after := PlaylistToBaseItem(p)

		// Finamp caches covers keyed by blurHash, so tag and blurhash must change with the cover.
		Expect(after.ImageTags["Primary"]).ToNot(Equal(before.ImageTags["Primary"]))
		Expect(after.ImageBlurHashes["Primary"]).ToNot(Equal(before.ImageBlurHashes["Primary"]))
	})

	It("keeps the playlist image tag stable when nothing changed", func() {
		p := model.Playlist{ID: "pl-1", UpdatedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
		Expect(PlaylistToBaseItem(p).ImageTags).To(Equal(PlaylistToBaseItem(p).ImageTags))
	})
})

var _ = Describe("LyricDtoFromLyrics", func() {
	ms := func(v int64) *int64 { return &v }

	mf := model.MediaFile{ID: "s1", Title: "Song", Artist: "Artist", Album: "Album", Duration: 100}

	It("maps synced lyrics with tick conversion", func() {
		l := model.Lyrics{
			DisplayArtist: "Display Artist",
			DisplayTitle:  "Display Title",
			Synced:        true,
			Offset:        ms(-150),
			Line: []model.Line{
				{Start: ms(1000), Value: "line one"},
				{Start: ms(2500), Value: "line two"},
			},
		}
		d := LyricDtoFromLyrics(mf, l)
		Expect(d.Metadata.Artist).To(Equal("Display Artist"))
		Expect(d.Metadata.Title).To(Equal("Display Title"))
		Expect(d.Metadata.Album).To(Equal("Album"))
		Expect(d.Metadata.IsSynced).To(BeTrue())
		Expect(*d.Metadata.Offset).To(Equal(int64(-1_500_000)))
		Expect(d.Metadata.Length).To(Equal(TicksFromSeconds(100)))
		Expect(d.Lyrics).To(HaveLen(2))
		Expect(d.Lyrics[0].Text).To(Equal("line one"))
		Expect(*d.Lyrics[0].Start).To(Equal(int64(10_000_000)))
		Expect(*d.Lyrics[1].Start).To(Equal(int64(25_000_000)))
	})

	It("falls back to the media file's artist and title", func() {
		d := LyricDtoFromLyrics(mf, model.Lyrics{Line: []model.Line{{Value: "x"}}})
		Expect(d.Metadata.Artist).To(Equal("Artist"))
		Expect(d.Metadata.Title).To(Equal("Song"))
	})

	It("drops start-less lines from synced lyrics", func() {
		l := model.Lyrics{Synced: true, Line: []model.Line{
			{Start: ms(0), Value: "kept"},
			{Value: "dropped"},
		}}
		d := LyricDtoFromLyrics(mf, l)
		Expect(d.Lyrics).To(HaveLen(1))
		Expect(d.Lyrics[0].Text).To(Equal("kept"))
	})

	It("emits no Start on unsynced lyrics even when lines have one", func() {
		l := model.Lyrics{Synced: false, Line: []model.Line{{Start: ms(1000), Value: "plain"}}}
		d := LyricDtoFromLyrics(mf, l)
		Expect(d.Lyrics).To(HaveLen(1))
		Expect(d.Lyrics[0].Start).To(BeNil())
		Expect(d.Metadata.IsSynced).To(BeFalse())
	})

	It("maps word cues", func() {
		end := int64(1500)
		l := model.Lyrics{Synced: true, Line: []model.Line{{
			Start: ms(1000),
			Value: "word cue",
			Cue:   []model.Cue{{Start: ms(1000), End: &end, Value: "word", ByteStart: 0, ByteEnd: 4}},
		}}}
		d := LyricDtoFromLyrics(mf, l)
		Expect(d.Lyrics[0].Cues).To(HaveLen(1))
		c := d.Lyrics[0].Cues[0]
		Expect(c.Position).To(Equal(0))
		Expect(c.EndPosition).To(Equal(4))
		Expect(c.Start).To(Equal(int64(10_000_000)))
		Expect(*c.End).To(Equal(int64(15_000_000)))
	})

	It("skips a start-less cue while keeping its sibling", func() {
		l := model.Lyrics{Synced: true, Line: []model.Line{{
			Start: ms(1000),
			Value: "word cue",
			Cue: []model.Cue{
				{Start: nil, Value: "dropped", ByteStart: 0, ByteEnd: 7},
				{Start: ms(1000), Value: "kept", ByteStart: 8, ByteEnd: 12},
			},
		}}}
		d := LyricDtoFromLyrics(mf, l)
		Expect(d.Lyrics[0].Cues).To(HaveLen(1))
		c := d.Lyrics[0].Cues[0]
		Expect(c.Position).To(Equal(8))
		Expect(c.EndPosition).To(Equal(12))
		Expect(c.Start).To(Equal(int64(10_000_000)))
	})
})
