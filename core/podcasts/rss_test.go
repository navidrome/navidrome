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
