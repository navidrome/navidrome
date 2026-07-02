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
//  3. Specificity level (0–5, based on metadata precision; higher is better)
//  4. Artist overlap (how many of the song's artists the track credits; more
//     shared artists is better)
//  5. Preferred-track flag (enabled by conf.Server.Matcher.PreferStarred;
//     prioritizes tracks that are starred or rated >= 4, but only among
//     candidates of equal specificity and overlap)
//  6. Album similarity (Jaro-Winkler, as the final tiebreaker)
//
// The specificity levels, from most to least specific, are:
//
//	Level 5: Title + Artist identity + Album MBID
//	Level 4: Title + Artist identity + Album name (fuzzy)
//	Level 3: Title + Artist name + Album name (fuzzy)
//	Level 2: Title + Artist identity
//	Level 1: Title + Artist name
//	Level 0: Title only
//
// "Artist identity" is a match on the artist's Navidrome ID (the strongest signal,
// when a source supplies one) or its MBID. A plain name match is the weaker fallback
// used for an artist with no identity match (e.g. a cover credited to a different
// artist of the same name).
//
// The title phase always requires an agent artist to scope the library query, so
// Level 0 does not mean "no artist": it applies when a candidate matches on title
// but its own artist differs from the query's (e.g. a cover or a featured-artist
// credit), leaving the title as the only shared field.
//
// A song may carry several artists, and the title phase scopes candidate tracks by
// ANY of them: a track credited to at least one shared artist is considered. When a
// source supplies a Navidrome artist ID, that artist is matched directly, skipping
// name/MBID resolution. Among equally specific candidates, the one sharing more of
// the song's artists wins, so a track crediting every collaborator outranks one
// crediting only a single artist.
//
// Each input song is scored independently, so two songs with the same title and
// artist but different durations can resolve to different library tracks (each
// matches the track closest to its own duration).
//
// # Examples
//
// All examples below exercise the title+artist phase, where the interesting
// behavior lives. (Identifier phases — ID, MBID, ISRC — are exact lookups that
// always win over fuzzy matching; they need no illustration.)
//
// Title threshold — a near-miss title still matches; an exact-only threshold
// rejects it:
//
//	Agent returns: {Name: "Bohemian Rhapsody", Artist: "Queen"}
//	Library has:   {ID: "t1", Title: "Bohemian Rhapsody - Remastered", Artist: "Queen"}
//	With threshold 85%: match succeeds (similarity ~0.87)
//	With threshold 100%: no match (not an exact title)
//
// Specificity ranking — among candidates that clear the title threshold, a
// better album match wins:
//
//	Agent returns: {Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}
//	Library has:
//	  {ID: "t1", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101"}       // Level 1
//	  {ID: "t2", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}  // Level 3
//	Result: t2 (Level 3 beats Level 1 on the album match)
//
// Duration tiebreak — with title and artist equal, the closest duration wins,
// so two near-identical input songs can resolve to different tracks:
//
//	Agent returns:
//	  {Name: "Untitled", Artist: "Interpol", Duration: 245000}  // 4:05
//	  {Name: "Untitled", Artist: "Interpol", Duration: 600000}  // 10:00 (a live take)
//	Library has:
//	  {ID: "studio", Title: "Untitled", Artist: "Interpol", Duration: 248}  // 4:08
//	  {ID: "live",   Title: "Untitled", Artist: "Interpol", Duration: 602}  // 10:02
//	Result: studio for the first song, live for the second
//
// Preferred track — when conf.Server.Matcher.PreferStarred is enabled, a
// starred (or rating >= 4) track is preferred, but only when specificity and
// artist overlap are equal. A more specific match always wins regardless of the
// preferred flag:
//
//	Agent returns: {Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}
//	Library has:
//	  {ID: "exact",   Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}            // Level 3
//	  {ID: "starred", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Singles", Starred: true} // Level 1, starred
//	Result: exact (specificity outranks the starred flag; preferred only breaks ties of equal identity)
package matcher
