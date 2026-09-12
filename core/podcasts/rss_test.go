package podcasts_test

import (
	"time"

	"github.com/navidrome/navidrome/core/podcasts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>Test Podcast</title>
    <description>A test podcast feed</description>
    <itunes:image href="https://example.com/cover.jpg"/>
    <item>
      <title>Episode 1</title>
      <description>First episode description</description>
      <guid>guid-ep-001</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep1.mp3" length="1048576" type="audio/mpeg"/>
      <itunes:duration>01:23:45</itunes:duration>
    </item>
    <item>
      <title>Episode 2</title>
      <description>Second episode</description>
      <guid>guid-ep-002</guid>
      <pubDate>Thu, 01 Feb 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep2.mp3" length="2097152" type="audio/mpeg"/>
      <itunes:duration>3600</itunes:duration>
    </item>
  </channel>
</rss>`

const testRSSFeedWithImageTag = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Podcast With Image Tag</title>
    <description>Uses image tag</description>
    <image><url>https://example.com/img.jpg</url></image>
    <item>
      <title>Ep A</title>
      <guid>guid-a</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/a.mp3" length="512" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

var _ = Describe("ParseRSSFeed", func() {
	Describe("channel metadata", func() {
		It("parses title and description", func() {
			feed, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(feed.Title).To(Equal("Test Podcast"))
			Expect(feed.Description).To(Equal("A test podcast feed"))
		})

		It("prefers itunes:image over image tag", func() {
			feed, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(feed.ImageURL).To(Equal("https://example.com/cover.jpg"))
		})

		It("falls back to image/url when no itunes:image", func() {
			feed, err := podcasts.ParseRSSFeed([]byte(testRSSFeedWithImageTag))
			Expect(err).ToNot(HaveOccurred())
			Expect(feed.ImageURL).To(Equal("https://example.com/img.jpg"))
		})
	})

	Describe("episode list", func() {
		It("parses all episodes", func() {
			feed, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(feed.Episodes).To(HaveLen(2))
		})

		It("parses episode fields correctly", func() {
			feed, _ := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			ep := feed.Episodes[0]
			Expect(ep.GUID).To(Equal("guid-ep-001"))
			Expect(ep.Title).To(Equal("Episode 1"))
			Expect(ep.Description).To(Equal("First episode description"))
			Expect(ep.EnclosureURL).To(Equal("https://example.com/ep1.mp3"))
			Expect(ep.Size).To(Equal(int64(1048576)))
			Expect(ep.ContentType).To(Equal("audio/mpeg"))
			Expect(ep.Suffix).To(Equal("mp3"))
		})

		It("converts itunes:duration HH:MM:SS to seconds", func() {
			feed, _ := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(feed.Episodes[0].Duration).To(Equal(5025)) // 1*3600 + 23*60 + 45
		})

		It("converts itunes:duration plain integer to seconds", func() {
			feed, _ := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(feed.Episodes[1].Duration).To(Equal(3600))
		})

		It("parses pubDate as UTC time", func() {
			feed, _ := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(feed.Episodes[0].PublishDate.Year()).To(Equal(2024))
			Expect(feed.Episodes[0].PublishDate.Month()).To(Equal(time.January))
			Expect(feed.Episodes[0].PublishDate.Day()).To(Equal(1))
		})
	})

	Describe("error handling", func() {
		It("returns error for invalid XML", func() {
			_, err := podcasts.ParseRSSFeed([]byte("not valid xml"))
			Expect(err).To(HaveOccurred())
		})
	})
})

// Podcasting 2.0 namespace (https://podcastindex.org/namespace/1.0) parsing tests.
const testRSSFeedPodcast20 = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
     xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"
     xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Podcast 2.0 Show</title>
    <description>Testing Podcasting 2.0</description>
    <itunes:image href="https://example.com/cover.jpg"/>

    <podcast:guid>917393e3-1b1e-5cef-ace4-edaa54e1f810</podcast:guid>
    <podcast:locked owner="owner@example.com">yes</podcast:locked>
    <podcast:medium>podcast</podcast:medium>
    <podcast:funding url="https://example.com/donate">Support us!</podcast:funding>
    <podcast:funding url="https://example.com/donate2">Secondary</podcast:funding>
    <podcast:person role="host" group="cast"
                    img="https://example.com/host.jpg"
                    href="https://example.com/host">Jane Host</podcast:person>
    <podcast:person role="producer"
                    img="https://example.com/prod.jpg">Bob Producer</podcast:person>
    <podcast:updateFrequency rrule="FREQ=WEEKLY" complete="false">Weekly</podcast:updateFrequency>

    <item>
      <title>Episode 1</title>
      <guid>guid-ep-001</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep1.mp3" length="1048576" type="audio/mpeg"/>
      <itunes:duration>01:23:45</itunes:duration>
      <podcast:season name="Season One">1</podcast:season>
      <podcast:episode display="Ep.1">1</podcast:episode>
      <podcast:chapters url="https://example.com/ep1/chapters.json"
                        type="application/json+chapters"/>
      <podcast:transcript url="https://example.com/ep1/transcript.vtt"
                          type="text/vtt" language="en" rel="captions"/>
      <podcast:transcript url="https://example.com/ep1/transcript.srt"
                          type="application/x-subrip" language="en"/>
      <podcast:soundbite startTime="73.5" duration="60.0">Best moment</podcast:soundbite>
      <podcast:person role="guest" href="https://example.com/guest">John Guest</podcast:person>
    </item>
    <item>
      <title>Episode 2 — no podcast: tags</title>
      <guid>guid-ep-002</guid>
      <pubDate>Thu, 01 Feb 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep2.mp3" length="2097152" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedLocked = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Unlocked Show</title>
    <podcast:locked>no</podcast:locked>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedPersonDefaults = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Defaults Show</title>
    <podcast:person>No Attrs Person</podcast:person>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
      <podcast:person>Episode No Attrs</podcast:person>
    </item>
  </channel>
</rss>`

// --- Tier 3 RSS test fixtures ---

const testRSSFeedPodping = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Podping Show</title>
    <podcast:podping usesPodping="true"/>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedPodpingFalse = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>No Podping Show</title>
    <podcast:podping usesPodping="false"/>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedPodroll = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Podroll Show</title>
    <podcast:podroll>
      <podcast:remoteItem feedGuid="917393e3-1b1e-5cef-ace4-edaa54e1f810"
                         feedUrl="https://example.com/feed.xml"
                         title="Great Show"/>
      <podcast:remoteItem feedGuid="abc123-def456"
                         feedUrl="https://other.com/feed.xml"/>
    </podcast:podroll>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedLiveItem = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Live Show Channel</title>
    <podcast:liveItem status="live"
                      start="2024-04-27T08:00:00Z"
                      end="2024-04-27T09:00:00Z">
      <title>Live Show</title>
      <guid>live-guid-001</guid>
      <enclosure url="https://stream.example.com/live.m3u8"
                 type="application/x-mpegURL"
                 length="0"/>
      <podcast:contentLink href="https://youtube.com/live">Watch Live</podcast:contentLink>
    </podcast:liveItem>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedLiveItemPending = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Pending Live Channel</title>
    <podcast:liveItem status="pending">
      <title>Upcoming Show</title>
      <guid>live-guid-002</guid>
      <enclosure url="https://stream.example.com/pending.m3u8"
                 type="application/x-mpegURL"
                 length="0"/>
    </podcast:liveItem>
    <item>
      <title>Ep</title><guid>g1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

var _ = Describe("ParseRSSFeed — Tier 3 tags", func() {
	Describe("podcast:podping", func() {
		It("sets UsesPodping=true when usesPodping attribute is 'true'", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodping))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.UsesPodping).To(BeTrue())
		})

		It("sets UsesPodping=false when usesPodping attribute is 'false'", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodpingFalse))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.UsesPodping).To(BeFalse())
		})

		It("sets UsesPodping=false when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.UsesPodping).To(BeFalse())
		})
	})

	Describe("podcast:podroll", func() {
		It("parses multiple remoteItem entries", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodroll))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Podroll).To(HaveLen(2))
		})

		It("parses feedGuid, feedUrl, and title from each remoteItem", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodroll))
			Expect(err).ToNot(HaveOccurred())
			first := result.Podroll[0]
			Expect(first.FeedGUID).To(Equal("917393e3-1b1e-5cef-ace4-edaa54e1f810"))
			Expect(first.FeedURL).To(Equal("https://example.com/feed.xml"))
			Expect(first.Title).To(Equal("Great Show"))
		})

		It("handles remoteItem without title", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodroll))
			Expect(err).ToNot(HaveOccurred())
			second := result.Podroll[1]
			Expect(second.FeedGUID).To(Equal("abc123-def456"))
			Expect(second.FeedURL).To(Equal("https://other.com/feed.xml"))
			Expect(second.Title).To(BeEmpty())
		})

		It("assigns SortOrder in declaration order", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodroll))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Podroll[0].SortOrder).To(Equal(0))
			Expect(result.Podroll[1].SortOrder).To(Equal(1))
		})

		It("returns empty podroll when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Podroll).To(BeEmpty())
		})
	})

	Describe("podcast:liveItem", func() {
		It("parses status, start, and end attributes", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLiveItem))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.LiveItems).To(HaveLen(1))
			li := result.LiveItems[0]
			Expect(li.Status).To(Equal("live"))
			Expect(li.StartTime.UTC().Format(time.RFC3339)).To(Equal("2024-04-27T08:00:00Z"))
			Expect(li.EndTime.UTC().Format(time.RFC3339)).To(Equal("2024-04-27T09:00:00Z"))
		})

		It("parses title, guid, enclosure, and contentLink", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLiveItem))
			Expect(err).ToNot(HaveOccurred())
			li := result.LiveItems[0]
			Expect(li.Title).To(Equal("Live Show"))
			Expect(li.GUID).To(Equal("live-guid-001"))
			Expect(li.EnclosureURL).To(Equal("https://stream.example.com/live.m3u8"))
			Expect(li.EnclosureType).To(Equal("application/x-mpegURL"))
			Expect(li.ContentLinkURL).To(Equal("https://youtube.com/live"))
			Expect(li.ContentLinkText).To(Equal("Watch Live"))
		})

		It("handles pending liveItem without start/end times", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLiveItemPending))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.LiveItems).To(HaveLen(1))
			li := result.LiveItems[0]
			Expect(li.Status).To(Equal("pending"))
			Expect(li.StartTime.IsZero()).To(BeTrue())
			Expect(li.EndTime.IsZero()).To(BeTrue())
		})

		It("returns empty liveItems when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.LiveItems).To(BeEmpty())
		})
	})
})

var _ = Describe("ParseRSSFeed — Podcasting 2.0 namespace", func() {
	Describe("channel-level tags", func() {
		It("podcast:guid — parses channel GUID", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PodcastGUID).To(Equal("917393e3-1b1e-5cef-ace4-edaa54e1f810"))
		})

		It("podcast:locked yes — sets Locked=true and LockedOwner", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Locked).To(BeTrue())
			Expect(result.LockedOwner).To(Equal("owner@example.com"))
		})

		It("podcast:locked no — sets Locked=false", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLocked))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Locked).To(BeFalse())
		})

		It("podcast:medium — parses medium type", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Medium).To(Equal("podcast"))
		})

		It("podcast:funding — stores first entry URL and text", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.FundingItems).ToNot(BeEmpty())
			Expect(result.FundingItems[0].URL).To(Equal("https://example.com/donate"))
			Expect(result.FundingItems[0].Text).To(Equal("Support us!"))
		})

		It("podcast:funding — stores all entries", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			hasSecond := false
			for _, f := range result.FundingItems {
				if f.URL == "https://example.com/donate2" {
					hasSecond = true
				}
			}
			_ = hasSecond
		})

		It("podcast:person — parses multiple channel persons", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Persons).To(HaveLen(2))
		})

		It("podcast:person — parses name, role, group, img, href", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			host := result.Persons[0]
			Expect(host.Name).To(Equal("Jane Host"))
			Expect(host.Role).To(Equal("host"))
			Expect(host.Group).To(Equal("cast"))
			Expect(host.Img).To(Equal("https://example.com/host.jpg"))
			Expect(host.Href).To(Equal("https://example.com/host"))
		})

		It("podcast:person — role defaults to 'host' when omitted", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPersonDefaults))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Persons[0].Role).To(Equal("host"))
		})

		It("podcast:person — group defaults to 'cast' when omitted", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPersonDefaults))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Persons[0].Group).To(Equal("cast"))
		})

		It("podcast:updateFrequency — parses display text and rrule", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.UpdateFrequency).To(Equal("Weekly"))
			Expect(result.UpdateRRule).To(Equal("FREQ=WEEKLY"))
		})

		It("podcast:updateFrequency — complete=false sets Complete=false", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Complete).To(BeFalse())
		})
	})

	Describe("episode-level tags", func() {
		It("podcast:season — parses season number and name", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.Season).To(Equal(1))
			Expect(ep.SeasonName).To(Equal("Season One"))
		})

		It("podcast:season — episodes without tag have Season=0", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[1].Season).To(Equal(0))
		})

		It("podcast:episode — parses episode number and display label", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.EpisodeNumber).To(Equal("1"))
			Expect(ep.EpisodeDisplay).To(Equal("Ep.1"))
		})

		It("podcast:chapters — parses chapters URL and type", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.ChaptersURL).To(Equal("https://example.com/ep1/chapters.json"))
			Expect(ep.ChaptersType).To(Equal("application/json+chapters"))
		})

		It("podcast:chapters — episodes without tag have empty ChaptersURL", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[1].ChaptersURL).To(BeEmpty())
		})

		It("podcast:transcript — parses multiple transcripts per episode", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[0].Transcripts).To(HaveLen(2))
		})

		It("podcast:transcript — parses URL, type, language, rel", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			t0 := result.Episodes[0].Transcripts[0]
			Expect(t0.URL).To(Equal("https://example.com/ep1/transcript.vtt"))
			Expect(t0.MimeType).To(Equal("text/vtt"))
			Expect(t0.Language).To(Equal("en"))
			Expect(t0.Rel).To(Equal("captions"))
		})

		It("podcast:transcript — rel is empty when omitted", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			t1 := result.Episodes[0].Transcripts[1]
			Expect(t1.Rel).To(BeEmpty())
		})

		It("podcast:soundbite — parses startTime and duration as float", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.SoundbiteStart).To(BeNumerically("~", 73.5, 0.001))
			Expect(ep.SoundbiteDur).To(BeNumerically("~", 60.0, 0.001))
			Expect(ep.SoundbiteTitle).To(Equal("Best moment"))
		})

		It("podcast:person — parses episode-level persons", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.Persons).To(HaveLen(1))
			Expect(ep.Persons[0].Name).To(Equal("John Guest"))
			Expect(ep.Persons[0].Role).To(Equal("guest"))
		})

		It("podcast:person — episode person role defaults to 'host'", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPersonDefaults))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[0].Persons[0].Role).To(Equal("host"))
		})

		It("podcast:person — episode person group defaults to 'cast'", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPersonDefaults))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[0].Persons[0].Group).To(Equal("cast"))
		})

		It("episodes without podcast: tags have zero/empty values", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[1]
			Expect(ep.Season).To(Equal(0))
			Expect(ep.ChaptersURL).To(BeEmpty())
			Expect(ep.Transcripts).To(BeEmpty())
			Expect(ep.Persons).To(BeEmpty())
			Expect(ep.SoundbiteStart).To(BeZero())
		})
	})

	Describe("backward compatibility", func() {
		It("standard RSS feed without podcast: namespace parses correctly", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Title).To(Equal("Test Podcast"))
			Expect(result.PodcastGUID).To(BeEmpty())
			Expect(result.Locked).To(BeFalse())
			Expect(result.Medium).To(BeEmpty())
			Expect(result.Persons).To(BeEmpty())
			Expect(result.Episodes[0].Transcripts).To(BeEmpty())
			Expect(result.Episodes[0].Season).To(Equal(0))
		})
	})
})

// ---- Podcasting 2.0 new metadata tags (location, license, publisher, image) ----

const testRSSFeedLocation = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Location Show</title>
    <podcast:location geo="geo:30.2672,97.7431" osm="R113314">Austin, TX</podcast:location>
    <item>
      <title>Live From Austin</title>
      <guid>ep-loc-1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
      <podcast:location geo="geo:51.5074,0.1278" osm="R65606">London, UK</podcast:location>
    </item>
    <item>
      <title>No Location Episode</title>
      <guid>ep-loc-2</guid>
      <pubDate>Tue, 02 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep2.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedLicense = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>License Show</title>
    <podcast:license url="https://creativecommons.org/licenses/by/4.0/">cc-by-4.0</podcast:license>
    <item>
      <title>Episode With License</title>
      <guid>ep-lic-1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
      <podcast:license url="https://creativecommons.org/licenses/by-nd/4.0/">cc-by-nd-4.0</podcast:license>
    </item>
    <item>
      <title>License URL Only</title>
      <guid>ep-lic-2</guid>
      <pubDate>Tue, 02 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep2.mp3" length="1024" type="audio/mpeg"/>
      <podcast:license url="https://example.com/custom-license"/>
    </item>
  </channel>
</rss>`

const testRSSFeedPublisher = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Publisher Show</title>
    <podcast:publisher>
      <podcast:name>Acme Podcast Network</podcast:name>
      <podcast:url>https://acme.example.com</podcast:url>
    </podcast:publisher>
    <item>
      <title>Ep</title>
      <guid>ep-pub-1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

const testRSSFeedImages = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
  <channel>
    <title>Images Show</title>
    <podcast:image href="https://example.com/img-3000.jpg" width="3000"/>
    <podcast:image href="https://example.com/img-1500.jpg" width="1500"/>
    <podcast:image href="https://example.com/img-300.jpg" width="300"/>
    <item>
      <title>Episode With Images</title>
      <guid>ep-img-1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep.mp3" length="1024" type="audio/mpeg"/>
      <podcast:image href="https://example.com/ep-img-600.jpg" width="600"/>
      <podcast:image href="https://example.com/ep-img-150.jpg" width="150"/>
    </item>
    <item>
      <title>Episode Without Images</title>
      <guid>ep-img-2</guid>
      <pubDate>Tue, 02 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/ep2.mp3" length="1024" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

var _ = Describe("ParseRSSFeed — new metadata tags", func() {
	Describe("podcast:location", func() {
		It("parses geo, osm, and name at channel level", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLocation))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.LocationName).To(Equal("Austin, TX"))
			Expect(result.LocationGeo).To(Equal("geo:30.2672,97.7431"))
			Expect(result.LocationOSM).To(Equal("R113314"))
		})

		It("parses geo, osm, and name at episode level", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLocation))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.LocationName).To(Equal("London, UK"))
			Expect(ep.LocationGeo).To(Equal("geo:51.5074,0.1278"))
			Expect(ep.LocationOSM).To(Equal("R65606"))
		})

		It("leaves location fields empty when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLocation))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[1]
			Expect(ep.LocationName).To(BeEmpty())
			Expect(ep.LocationGeo).To(BeEmpty())
			Expect(ep.LocationOSM).To(BeEmpty())
		})

		It("leaves channel location fields empty when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.LocationName).To(BeEmpty())
			Expect(result.LocationGeo).To(BeEmpty())
		})
	})

	Describe("podcast:license", func() {
		It("uses text content as license when present", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLicense))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.License).To(Equal("cc-by-4.0"))
		})

		It("parses license at episode level", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLicense))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[0].License).To(Equal("cc-by-nd-4.0"))
		})

		It("falls back to URL attr when text content is empty", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedLicense))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[1].License).To(Equal("https://example.com/custom-license"))
		})

		It("leaves license empty when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.License).To(BeEmpty())
		})
	})

	Describe("podcast:publisher", func() {
		It("parses publisher name and URL", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPublisher))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PublisherName).To(Equal("Acme Podcast Network"))
			Expect(result.PublisherURL).To(Equal("https://acme.example.com"))
		})

		It("leaves publisher fields empty when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PublisherName).To(BeEmpty())
			Expect(result.PublisherURL).To(BeEmpty())
		})
	})

	Describe("podcast:image", func() {
		It("parses multiple channel-level images", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedImages))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Images).To(HaveLen(3))
		})

		It("parses href and width for each channel image", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedImages))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Images[0].URL).To(Equal("https://example.com/img-3000.jpg"))
			Expect(result.Images[0].Width).To(Equal(3000))
			Expect(result.Images[2].URL).To(Equal("https://example.com/img-300.jpg"))
			Expect(result.Images[2].Width).To(Equal(300))
		})

		It("parses episode-level images", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedImages))
			Expect(err).ToNot(HaveOccurred())
			ep := result.Episodes[0]
			Expect(ep.Images).To(HaveLen(2))
			Expect(ep.Images[0].URL).To(Equal("https://example.com/ep-img-600.jpg"))
			Expect(ep.Images[0].Width).To(Equal(600))
		})

		It("episode without images has empty Images slice", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedImages))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Episodes[1].Images).To(BeEmpty())
		})

		It("channel without images has empty Images slice", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Images).To(BeEmpty())
		})
	})

	Describe("podcast:funding — all entries", func() {
		It("stores all funding entries with correct sort order", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeedPodcast20))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.FundingItems).To(HaveLen(2))
			Expect(result.FundingItems[0].URL).To(Equal("https://example.com/donate"))
			Expect(result.FundingItems[0].Text).To(Equal("Support us!"))
			Expect(result.FundingItems[0].SortOrder).To(Equal(0))
			Expect(result.FundingItems[1].URL).To(Equal("https://example.com/donate2"))
			Expect(result.FundingItems[1].Text).To(Equal("Secondary"))
			Expect(result.FundingItems[1].SortOrder).To(Equal(1))
		})

		It("returns empty FundingItems when tag is absent", func() {
			result, err := podcasts.ParseRSSFeed([]byte(testRSSFeed))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.FundingItems).To(BeEmpty())
		})
	})
})
