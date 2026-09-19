package podcasts

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/server/events"
)

const podcastLibraryName = "Podcasts"

type Podcasts interface {
	AddChannel(ctx context.Context, rssURL string) error
	RefreshChannels(ctx context.Context) error
	DeleteChannel(ctx context.Context, id string) error
	DeleteEpisode(ctx context.Context, id string) error
	DownloadEpisode(ctx context.Context, id string) error
}

type podcastService struct {
	rootCtx context.Context
	ds      model.DataStore
	ff      ffmpeg.FFmpeg
	broker  events.Broker
}

func NewPodcastService(rootCtx context.Context, ds model.DataStore, ff ffmpeg.FFmpeg, broker events.Broker) Podcasts {
	return &podcastService{rootCtx: rootCtx, ds: ds, ff: ff, broker: broker}
}

// podcastLibraryID returns the ID of the podcast virtual library,
// creating it if it doesn't exist. The library root is DataFolder so that
// MediaFile paths stored as "podcasts/{ch}/{ep}.mp3" resolve correctly via AbsolutePath().
func (s *podcastService) podcastLibraryID(ctx context.Context) (int, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	for _, lib := range libs {
		if lib.Name == podcastLibraryName {
			return lib.ID, nil
		}
	}
	lib := &model.Library{
		Name: podcastLibraryName,
		Path: conf.Server.DataFolder.String(),
	}
	if err := s.ds.Library(ctx).Put(lib); err != nil {
		return 0, err
	}
	return lib.ID, nil
}

func (s *podcastService) AddChannel(ctx context.Context, rssURL string) error {
	exists, err := s.ds.PodcastChannel(ctx).ExistsByURL(rssURL)
	if err != nil {
		return fmt.Errorf("checking existing channel: %w", err)
	}
	if exists {
		return fmt.Errorf("channel already exists: %s", rssURL)
	}

	feed, err := fetchAndParse(rssURL)
	if err != nil {
		return fmt.Errorf("adding podcast channel: %w", err)
	}

	ch := &model.PodcastChannel{
		URL:             rssURL,
		Title:           feed.Title,
		Description:     feed.Description,
		ImageURL:        feed.ImageURL,
		Status:          model.PodcastStatusNew,
		PodcastGUID:     feed.PodcastGUID,
		Locked:          feed.Locked,
		LockedOwner:     feed.LockedOwner,
		Medium:          feed.Medium,
		UpdateFrequency: feed.UpdateFrequency,
		UpdateRRule:     feed.UpdateRRule,
		Complete:        feed.Complete,
		UsesPodping:     feed.UsesPodping,
		LocationName:    feed.LocationName,
		LocationGeo:     feed.LocationGeo,
		LocationOSM:     feed.LocationOSM,
		License:         feed.License,
		PublisherName:   feed.PublisherName,
		PublisherURL:    feed.PublisherURL,
	}
	if err := s.ds.PodcastChannel(ctx).Create(ch); err != nil {
		return err
	}

	// Save channel-level persons
	if len(feed.Persons) > 0 {
		if err := s.ds.PodcastPerson(ctx).SaveForChannel(ch.ID, feed.Persons); err != nil {
			log.Warn(ctx, "Failed to save podcast channel persons", "channel", ch.ID, err)
		}
	}

	// Save podcast:funding items
	if len(feed.FundingItems) > 0 {
		for i := range feed.FundingItems {
			feed.FundingItems[i].ChannelID = ch.ID
		}
		if err := s.ds.PodcastFunding(ctx).SaveForChannel(ch.ID, feed.FundingItems); err != nil {
			log.Warn(ctx, "Failed to save podcast funding items", "channel", ch.ID, err)
		}
	}

	// Save podcast:image (channel level)
	if len(feed.Images) > 0 {
		if err := s.ds.PodcastImage(ctx).SaveForChannel(ch.ID, feed.Images); err != nil {
			log.Warn(ctx, "Failed to save podcast channel images", "channel", ch.ID, err)
		}
	}

	// Save podcast:podroll items
	if len(feed.Podroll) > 0 {
		if err := s.ds.PodcastPodroll(ctx).SaveForChannel(ch.ID, feed.Podroll); err != nil {
			log.Warn(ctx, "Failed to save podcast podroll", "channel", ch.ID, err)
		}
	}

	// Save podcast:liveItem entries
	for _, li := range feed.LiveItems {
		li.ChannelID = ch.ID
		if err := s.ds.PodcastLiveItem(ctx).Upsert(&li); err != nil {
			log.Warn(ctx, "Failed to save podcast live item", "channel", ch.ID, err)
		}
	}

	for i := range feed.Episodes {
		ep := feed.Episodes[i]
		ep.ChannelID = ch.ID
		ep.Status = model.PodcastStatusNew
		transcripts := ep.Transcripts
		persons := ep.Persons
		images := ep.Images
		ep.Transcripts = nil
		ep.Persons = nil
		ep.Images = nil
		if err := s.ds.PodcastEpisode(ctx).Create(&ep); err != nil {
			return err
		}
		// Save episode transcripts
		if len(transcripts) > 0 {
			for j := range transcripts {
				transcripts[j].EpisodeID = ep.ID
			}
			if err := s.ds.PodcastTranscript(ctx).Save(transcripts); err != nil {
				log.Warn(ctx, "Failed to save podcast episode transcripts", "episode", ep.ID, err)
			}
		}
		// Save episode persons
		if len(persons) > 0 {
			if err := s.ds.PodcastPerson(ctx).SaveForEpisode(ep.ID, persons); err != nil {
				log.Warn(ctx, "Failed to save podcast episode persons", "episode", ep.ID, err)
			}
		}
		// Save episode images
		if len(images) > 0 {
			if err := s.ds.PodcastImage(ctx).SaveForEpisode(ep.ID, images); err != nil {
				log.Warn(ctx, "Failed to save podcast episode images", "episode", ep.ID, err)
			}
		}
	}

	ch.Status = model.PodcastStatusCompleted
	return s.ds.PodcastChannel(ctx).UpdateChannel(ch)
}

func (s *podcastService) RefreshChannels(ctx context.Context) error {
	channels, err := s.ds.PodcastChannel(ctx).GetAll(false)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if ch.UsesPodping {
			continue // skip — this channel uses Podping for updates
		}
		if err := s.refreshChannel(ctx, ch); err != nil {
			log.Warn(ctx, "Failed to refresh podcast channel", "channel", ch.Title, err)
		}
	}
	return nil
}

func (s *podcastService) refreshChannel(ctx context.Context, ch model.PodcastChannel) error {
	feed, err := fetchAndParse(ch.URL)
	if err != nil {
		return err
	}

	// Refresh podcast:funding items
	if err := s.ds.PodcastFunding(ctx).SaveForChannel(ch.ID, feed.FundingItems); err != nil {
		log.Warn(ctx, "Failed to refresh funding items", "channel", ch.ID, err)
	}

	// Refresh podcast:image (channel level)
	if err := s.ds.PodcastImage(ctx).SaveForChannel(ch.ID, feed.Images); err != nil {
		log.Warn(ctx, "Failed to refresh channel images", "channel", ch.ID, err)
	}

	// Refresh podcast:podroll
	if err := s.ds.PodcastPodroll(ctx).SaveForChannel(ch.ID, feed.Podroll); err != nil {
		log.Warn(ctx, "Failed to refresh podroll", "channel", ch.ID, err)
	}

	// Refresh podcast:liveItem
	for _, li := range feed.LiveItems {
		li.ChannelID = ch.ID
		if err := s.ds.PodcastLiveItem(ctx).Upsert(&li); err != nil {
			log.Warn(ctx, "Failed to upsert live item", "channel", ch.ID, err)
		}
	}

	epRepo := s.ds.PodcastEpisode(ctx)
	for i := range feed.Episodes {
		ep := feed.Episodes[i]
		_, err := epRepo.GetByGUID(ch.ID, ep.GUID)
		if err == nil {
			continue // already exists
		}
		ep.ChannelID = ch.ID
		ep.Status = model.PodcastStatusNew
		transcripts := ep.Transcripts
		persons := ep.Persons
		images := ep.Images
		ep.Transcripts = nil
		ep.Persons = nil
		ep.Images = nil
		if err := epRepo.Create(&ep); err != nil {
			return err
		}
		// Save episode transcripts
		if len(transcripts) > 0 {
			for j := range transcripts {
				transcripts[j].EpisodeID = ep.ID
			}
			if err := s.ds.PodcastTranscript(ctx).Save(transcripts); err != nil {
				log.Warn(ctx, "Failed to save podcast episode transcripts", "episode", ep.ID, err)
			}
		}
		// Save episode persons
		if len(persons) > 0 {
			if err := s.ds.PodcastPerson(ctx).SaveForEpisode(ep.ID, persons); err != nil {
				log.Warn(ctx, "Failed to save podcast episode persons", "episode", ep.ID, err)
			}
		}
		// Save episode images
		if len(images) > 0 {
			if err := s.ds.PodcastImage(ctx).SaveForEpisode(ep.ID, images); err != nil {
				log.Warn(ctx, "Failed to save podcast episode images", "episode", ep.ID, err)
			}
		}
	}
	return nil
}

func (s *podcastService) DownloadEpisode(ctx context.Context, id string) error {
	ep, err := s.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}
	ch, err := s.ds.PodcastChannel(ctx).Get(ep.ChannelID)
	if err != nil {
		return err
	}

	ep.Status = model.PodcastStatusDownloading
	ep.UpdatedAt = time.Now()
	if err := s.ds.PodcastEpisode(ctx).Update(ep); err != nil {
		return err
	}

	go s.doDownload(s.rootCtx, ep, ch)
	return nil
}

func (s *podcastService) doDownload(ctx context.Context, ep *model.PodcastEpisode, ch *model.PodcastChannel) {
	suffix := ep.Suffix
	if suffix == "" {
		suffix = "mp3"
	}
	dir := filepath.Join(conf.Server.DataFolder.String(), "podcasts", ep.ChannelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.setEpisodeError(ctx, ep, err)
		return
	}

	dest := filepath.Join(dir, ep.ID+"."+suffix)
	f, err := os.Create(dest)
	if err != nil {
		s.setEpisodeError(ctx, ep, err)
		return
	}
	defer f.Close()

	if err := validateURL(ep.EnclosureURL); err != nil {
		s.setEpisodeError(ctx, ep, fmt.Errorf("invalid enclosure URL: %w", err))
		return
	}
	httpClient := &http.Client{Timeout: 30 * time.Second, Transport: safeHTTPTransport}
	resp, err := httpClient.Get(ep.EnclosureURL) //nolint:gosec
	if err != nil {
		s.setEpisodeError(ctx, ep, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, ep.EnclosureURL)
		s.setEpisodeError(ctx, ep, err)
		return
	}

	// Use Content-Length as total size when RSS feed didn't provide it
	if resp.ContentLength > 0 && ep.Size == 0 {
		ep.Size = resp.ContentLength
	}

	size, err := io.Copy(&progressWriter{ep: ep, ds: s.ds, broker: s.broker, ctx: ctx, w: f}, resp.Body)
	if err != nil {
		s.setEpisodeError(ctx, ep, err)
		return
	}
	f.Close()

	// Write ID3 tags so the scanner picks up the correct metadata
	s.writeID3Tags(ctx, dest, suffix, ep.Title, ch.Title)

	// Register as a MediaFile so /rest/stream works with the standard media file path.
	// Use a podcast virtual library whose root is DataFolder; store relative path.
	libID, libErr := s.podcastLibraryID(ctx)
	if libErr != nil {
		log.Warn(ctx, "Failed to get podcast library, streaming may not work", "episode", ep.ID, libErr)
	} else {
		relPath := strings.TrimPrefix(dest, conf.Server.DataFolder.String()+string(filepath.Separator))
		now := time.Now()
		tags := model.Tags{}
		tags.Add("genre", "Podcast")
		mf := &model.MediaFile{
			ID:          id.NewRandom(),
			LibraryID:   libID,
			Path:        relPath,
			Title:       ep.Title,
			Album:       ch.Title,
			AlbumID:     ch.ID,
			Artist:      "",
			AlbumArtist: ch.Title,
			Genre:       "Podcast",
			Tags:        tags,
			Duration:    float32(ep.Duration),
			Size:        size,
			BitRate:     ep.BitRate,
			Suffix:      suffix,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if putErr := s.ds.MediaFile(ctx).Put(mf); putErr != nil {
			log.Warn(ctx, "Failed to register podcast episode as MediaFile", "episode", ep.ID, putErr)
		} else {
			ep.StreamID = mf.ID
		}
	}

	// Probe actual duration and bitrate from the downloaded file
	if s.ff != nil && s.ff.IsProbeAvailable() {
		if probe, probeErr := s.ff.ProbeAudioStream(ctx, dest); probeErr == nil {
			if probe.Duration > 0 {
				ep.Duration = int(math.Round(probe.Duration))
			}
			if probe.BitRate > 0 {
				ep.BitRate = probe.BitRate
			}
		} else {
			log.Warn(ctx, "Failed to probe podcast episode duration", "episode", ep.ID, probeErr)
		}
	}

	ep.Path = dest
	ep.Size = size
	ep.DownloadedBytes = size
	ep.Status = model.PodcastStatusCompleted
	ep.ErrorMessage = ""
	ep.UpdatedAt = time.Now()
	if err := s.ds.PodcastEpisode(ctx).Update(ep); err != nil {
		log.Error(ctx, "Failed to update episode after download", "episode", ep.ID, err)
	}
	if s.broker != nil {
		s.broker.SendBroadcastMessage(ctx, &events.PodcastEpisodeProgress{
			EpisodeID:       ep.ID,
			ChannelID:       ep.ChannelID,
			DownloadedBytes: size,
			Size:            size,
			Duration:        ep.Duration,
			Status:          string(model.PodcastStatusCompleted),
		})
	}
}

func (s *podcastService) setEpisodeError(ctx context.Context, ep *model.PodcastEpisode, err error) {
	ep.Status = model.PodcastStatusError
	ep.ErrorMessage = err.Error()
	ep.UpdatedAt = time.Now()
	if updateErr := s.ds.PodcastEpisode(ctx).Update(ep); updateErr != nil {
		log.Error(ctx, "Failed to set episode error status", "episode", ep.ID, updateErr)
	}
	if s.broker != nil {
		s.broker.SendBroadcastMessage(ctx, &events.PodcastEpisodeProgress{
			EpisodeID: ep.ID,
			ChannelID: ep.ChannelID,
			Status:    string(model.PodcastStatusError),
		})
	}
}

// sanitizeMetadata removes null bytes and trims whitespace from ffmpeg metadata values.
// Since exec.Command passes args directly (no shell), only null bytes need sanitizing.
func sanitizeMetadata(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\x00", "")
}

func (s *podcastService) writeID3Tags(ctx context.Context, dest, suffix, title, album string) {
	if s.ff == nil {
		return
	}
	ffmpegPath, err := s.ff.CmdPath()
	if err != nil {
		return
	}
	tmp := dest + ".tmp." + suffix
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", dest,
		"-metadata", "title="+sanitizeMetadata(title),
		"-metadata", "album="+sanitizeMetadata(album),
		"-metadata", "artist=",
		"-metadata", "genre=Podcast",
		"-c", "copy", "-y", tmp,
	)
	if err := cmd.Run(); err != nil {
		log.Warn(ctx, "Failed to write ID3 tags to podcast episode", "episode", dest, err)
		_ = os.Remove(tmp)
		return
	}
	if err := os.Rename(tmp, dest); err != nil {
		log.Warn(ctx, "Failed to replace podcast file with tagged version", err)
		_ = os.Remove(tmp)
	}
}

func (s *podcastService) DeleteEpisode(ctx context.Context, id string) error {
	ep, err := s.ds.PodcastEpisode(ctx).Get(id)
	if err != nil {
		return err
	}
	if ep.Path != "" {
		_ = os.Remove(ep.Path)
		ep.Path = ""
	}
	// Remove the registered MediaFile so it can be re-registered on next download
	if ep.StreamID != "" {
		_ = s.ds.MediaFile(ctx).Delete(ep.StreamID)
		ep.StreamID = ""
	}
	ep.Status = model.PodcastStatusNew
	ep.ErrorMessage = ""
	ep.Size = 0
	ep.DownloadedBytes = 0
	ep.Duration = 0
	ep.BitRate = 0
	ep.UpdatedAt = time.Now()
	return s.ds.PodcastEpisode(ctx).Update(ep)
}

func (s *podcastService) DeleteChannel(ctx context.Context, id string) error {
	episodes, err := s.ds.PodcastEpisode(ctx).GetByChannel(id)
	if err != nil {
		return err
	}
	for _, ep := range episodes {
		if ep.Path != "" {
			_ = os.Remove(ep.Path)
		}
	}
	return s.ds.PodcastChannel(ctx).Delete(id)
}

// progressWriter wraps an io.Writer and periodically saves download progress to DB.
type progressWriter struct {
	ep      *model.PodcastEpisode
	ds      model.DataStore
	broker  events.Broker
	ctx     context.Context
	w       io.Writer
	written int64
	lastDB  int64
}

const progressUpdateInterval = 512 * 1024 // update DB every 512 KB

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	pw.written += int64(n)
	if pw.written-pw.lastDB >= progressUpdateInterval {
		pw.ep.DownloadedBytes = pw.written
		pw.ep.UpdatedAt = time.Now()
		_ = pw.ds.PodcastEpisode(pw.ctx).Update(pw.ep)
		pw.lastDB = pw.written
		if pw.broker != nil {
			pw.broker.SendBroadcastMessage(pw.ctx, &events.PodcastEpisodeProgress{
				EpisodeID:       pw.ep.ID,
				ChannelID:       pw.ep.ChannelID,
				DownloadedBytes: pw.written,
				Size:            pw.ep.Size,
			})
		}
	}
	return n, err
}

// validateURL rejects any URL that is not a plain http/https request to a
// named host. It exists to prevent SSRF: without it, an authenticated user
// could point the preview/download endpoints at internal services, cloud
// metadata endpoints (e.g. 169.254.169.254), or any other host only
// reachable from the server itself. This is a cheap, fast-failing check on
// the URL's shape - the actual IP-level check happens per-connection in
// safeHTTPTransport below, since the host a URL names and the IP it
// resolves to at request time aren't guaranteed to be the same thing.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q, only http/https are allowed", u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("URL has no host")
	}
	return nil
}

// isReservedIP reports whether ip is a loopback, private, link-local,
// multicast, or otherwise non-routable/internal address. Cloud metadata
// endpoints (e.g. AWS/GCP/Azure's 169.254.169.254) fall under the
// link-local range, so they're covered without a special case.
//
// This is a var, not a plain func, only so AllowLoopbackHTTPForTests (below)
// can narrow it for test binaries - production code never reassigns it.
var isReservedIP = func(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// AllowLoopbackHTTPForTests relaxes safeHTTPTransport's SSRF guard to permit
// loopback addresses (127.0.0.0/8, ::1) - every other reserved/private/
// link-local range (including cloud metadata endpoints) is still refused.
// It exists because httptest.Server always binds to loopback, so the podcast
// test suite needs a way to point the service at one without disabling the
// guard entirely. Not for production use.
func AllowLoopbackHTTPForTests() {
	strict := isReservedIP
	isReservedIP = func(ip net.IP) bool {
		if ip.IsLoopback() {
			return false
		}
		return strict(ip)
	}
}

// safeHTTPTransport is shared by every outbound podcast HTTP request (RSS
// feed fetch and episode download). Its DialContext resolves the host and
// checks isReservedIP at the moment of connection, not just once via
// validateURL up front - so a DNS answer that changes between the URL
// check and the actual TCP connect (DNS rebinding) can't be used to reach
// a reserved address that validateURL alone would have caught.
var safeHTTPTransport = &http.Transport{
	DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("parsing address %q: %w", addr, err)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolving host %q: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("host %q did not resolve to any address", host)
		}
		var dialer net.Dialer
		var lastErr error
		for _, ip := range ips {
			if isReservedIP(ip.IP) {
				lastErr = fmt.Errorf("host %q resolves to a reserved/internal address (%s), refusing to connect", host, ip.IP)
				continue
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, lastErr
	},
}

func fetchAndParse(rssURL string) (*rssFeed, error) {
	if err := validateURL(rssURL); err != nil {
		return nil, fmt.Errorf("invalid RSS feed URL: %w", err)
	}
	httpClient := &http.Client{Timeout: 15 * time.Second, Transport: safeHTTPTransport}
	resp, err := httpClient.Get(rssURL) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("fetching RSS feed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading RSS feed: %w", err)
	}

	return ParseRSSFeed(data)
}
