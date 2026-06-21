// Package matcher matches song results from external agents (Last.fm, Deezer,
// etc.) to tracks in the local music library, prioritizing accuracy over recall.
//
// It exposes a single [Matcher] type with two entry points that share the same
// matching algorithm:
//
//   - [Matcher.MatchSongs] returns an ordered, deduplicated slice of library
//     tracks, capped at a requested count. Use it when presenting "similar
//     songs" results to a client.
//   - [Matcher.MatchSongsIndexed] returns a map from input-song index to matched
//     track, with no deduplication. Use it when the caller needs to correlate
//     each result back to its input position (e.g. to attach a per-song
//     similarity score).
//
// # Algorithm Overview
//
// Each input song is resolved to its best-matching library track using four
// strategies, applied in priority order. A song matched by a higher-priority
// strategy is never reconsidered by a lower-priority one:
//
//  1. Direct ID match: songs with an ID are matched to a MediaFile by ID.
//  2. MusicBrainz Recording ID (MBID) match: songs with an MBID are matched to
//     tracks with the same mbz_recording_id.
//  3. ISRC match: songs with an ISRC are matched to tracks carrying that ISRC tag.
//  4. Title+Artist fuzzy match: remaining songs are matched by fuzzy string
//     comparison with metadata-specificity scoring (see below).
//
// Priority order is ID > MBID > ISRC > Title+Artist, so more reliable
// identifiers always take precedence over fuzzy text matching. Missing tracks
// (those no longer present on disk) are never matched.
//
// # Fuzzy Matching Details
//
// Title+artist matching uses Jaro-Winkler similarity, with a threshold
// configurable via conf.Server.Matcher.FuzzyThreshold (default 85%). A library
// track must clear the title threshold to be considered. Candidates that clear
// it are ranked by, in order:
//
//  1. Title similarity (Jaro-Winkler score, 0.0–1.0)
//  2. Duration proximity (closer duration scores higher; 1.0 when the agent
//     reports no duration)
//  3. Preferred-track flag (enabled by conf.Server.Matcher.PreferStarred;
//     prioritizes tracks that are starred or rated >= 4)
//  4. Specificity level (0–5, based on metadata precision; higher is better)
//  5. Album similarity (Jaro-Winkler, as the final tiebreaker)
//
// The specificity levels, from most to least specific, are:
//
//	Level 5: Title + Artist MBID + Album MBID
//	Level 4: Title + Artist MBID + Album name (fuzzy)
//	Level 3: Title + Artist name + Album name (fuzzy)
//	Level 2: Title + Artist MBID
//	Level 1: Title + Artist name
//	Level 0: Title only
//
// Each input song is scored independently, so two songs with the same title and
// artist but different durations can resolve to different library tracks (each
// matches the track closest to its own duration).
//
// # Examples
//
// MBID priority — an identifier match wins over a title+artist match:
//
//	Agent returns: {Name: "Paranoid Android", MBID: "abc-123", Artist: "Radiohead"}
//	Library has:
//	  {ID: "t1", Title: "Paranoid Android", MbzRecordingID: "abc-123"}
//	  {ID: "t2", Title: "Paranoid Android", Artist: "Radiohead"}
//	Result: t1 (MBID match takes priority over title+artist)
//
// ISRC priority — likewise, an ISRC match wins over title+artist:
//
//	Agent returns: {Name: "Paranoid Android", ISRC: "GBAYE0000351", Artist: "Radiohead"}
//	Library has:
//	  {ID: "t1", Title: "Paranoid Android", Tags: {isrc: ["GBAYE0000351"]}}
//	  {ID: "t2", Title: "Paranoid Android", Artist: "Radiohead"}
//	Result: t1 (ISRC match takes priority over title+artist)
//
// Specificity ranking — a better album match wins among title+artist candidates:
//
//	Agent returns: {Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}
//	Library has:
//	  {ID: "t1", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101"}       // Level 1
//	  {ID: "t2", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}  // Level 3
//	Result: t2 (Level 3 beats Level 1 due to the album match)
//
// Fuzzy title matching — the threshold controls how close a title must be:
//
//	Agent returns: {Name: "Bohemian Rhapsody", Artist: "Queen"}
//	Library has:   {ID: "t1", Title: "Bohemian Rhapsody - Remastered", Artist: "Queen"}
//	With threshold 85%: match succeeds (similarity ~0.87)
//	With threshold 100%: no match (not an exact title)
package matcher
