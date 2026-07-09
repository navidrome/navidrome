$file = "C:\Development\navidrome-pulse\pulse-plugin\main.go"
$content = Get-Content $file -Raw

$oldScrobble = @'
func (p *pulsePlugin) Scrobble(req scrobbler.ScrobbleRequest) error {
        if req.Track.ID == "" {
                return nil
        }
        username := resolveUsername(req.Username)
        artistID := ""
        if len(req.Track.Artists) > 0 {
                artistID = req.Track.Artists[0].ID
        }
        pdk.Log(pdk.LogInfo, fmt.Sprintf("Pulse: Scrobble user=%q title=%q album=%q", username, req.Track.Title, req.Track.Album))
        writePlayLog(username, PlayLogEntry{
                UnixTS:    req.Timestamp,
                TrackID:   req.Track.ID,
                TrackName: req.Track.Title,
                Artist:    req.Track.Artist,
                ArtistID:  artistID,
                Album:     req.Track.Album,
                Duration:  req.Track.Duration,
                Username:  req.Username,
        })
        return nil
}
'@

$newScrobble = @'
func (p *pulsePlugin) Scrobble(req scrobbler.ScrobbleRequest) error {
	if req.Track.ID == "" {
		return nil
	}
	username := resolveUsername(req.Username)
	artistID := ""
	if len(req.Track.Artists) > 0 {
		artistID = req.Track.Artists[0].ID
	}
	// --- ENHANCED LOGGING ---
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Pulse: Scrobble user=%q client=%q source=%q origin=%q mode=%q title=%q",
		username, req.Client, req.Source, req.Origin, req.PlaybackMode, req.Track.Title))
	// ------------------------

	writePlayLog(username, PlayLogEntry{
		UnixTS:    req.Timestamp,
		TrackID:   req.Track.ID,
		TrackName: req.Track.Title,
		Artist:    req.Track.Artist,
		ArtistID:  artistID,
		Album:     req.Track.Album,
		Duration:  req.Track.Duration,
		Username:  req.Username,
	})
	return nil
}
'@

$content = $content.Replace($oldScrobble, $newScrobble)
$content | Out-File $file -Encoding utf8
