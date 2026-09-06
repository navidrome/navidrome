//! Maps Lofty tag maps into scan-ready MediaFile JSON for the Go scanner hot path.

use std::collections::{HashMap, HashSet};
use std::path::Path;

use serde::Serialize;

use crate::tag_clean;

const DEFAULT_ARTISTS_SPLIT: &[&str] = &[" / ", " feat. ", " feat ", " ft. ", " ft ", "; "];
const DEFAULT_ROLES_SPLIT: &[&str] = &["/", ";"];
const DEFAULT_ARTIST_JOINER: &str = " • ";

/// Participant-splitting options mirrored from mappings.yaml and Scanner config.
#[derive(Debug, Clone)]
pub struct MapMediaConfig {
    pub artists_split: Vec<String>,
    pub roles_split: Vec<String>,
    pub artist_split_exceptions: Vec<String>,
    pub artist_joiner: String,
}

impl Default for MapMediaConfig {
    fn default() -> Self {
        Self::with_defaults()
    }
}

impl MapMediaConfig {
    pub fn with_defaults() -> Self {
        Self {
            artists_split: DEFAULT_ARTISTS_SPLIT.iter().map(|s| (*s).to_owned()).collect(),
            roles_split: DEFAULT_ROLES_SPLIT.iter().map(|s| (*s).to_owned()).collect(),
            artist_split_exceptions: Vec::new(),
            artist_joiner: DEFAULT_ARTIST_JOINER.to_owned(),
        }
    }
}

#[derive(Debug, Serialize)]
struct ScanArtist {
    name: String,
    #[serde(rename = "sortArtistName", skip_serializing_if = "String::is_empty")]
    sort_artist_name: String,
    #[serde(rename = "orderArtistName", skip_serializing_if = "String::is_empty")]
    order_artist_name: String,
    #[serde(rename = "mbzArtistId", skip_serializing_if = "String::is_empty")]
    mbz_artist_id: String,
    #[serde(rename = "subRole", skip_serializing_if = "String::is_empty")]
    sub_role: String,
}

#[derive(Debug, Serialize)]
struct ScanMediaFile {
    title: String,
    album: String,
    artist: String,
    #[serde(rename = "albumArtist")]
    album_artist: String,
    #[serde(rename = "sortTitle", skip_serializing_if = "String::is_empty")]
    sort_title: String,
    #[serde(rename = "sortAlbumName", skip_serializing_if = "String::is_empty")]
    sort_album_name: String,
    #[serde(rename = "sortArtistName", skip_serializing_if = "String::is_empty")]
    sort_artist_name: String,
    #[serde(rename = "sortAlbumArtistName", skip_serializing_if = "String::is_empty")]
    sort_album_artist_name: String,
    #[serde(rename = "orderTitle", skip_serializing_if = "String::is_empty")]
    order_title: String,
    #[serde(rename = "orderAlbumName", skip_serializing_if = "String::is_empty")]
    order_album_name: String,
    compilation: bool,
    #[serde(rename = "trackNumber")]
    track_number: i32,
    #[serde(rename = "discNumber")]
    disc_number: i32,
    #[serde(rename = "discSubtitle", skip_serializing_if = "String::is_empty")]
    disc_subtitle: String,
    #[serde(rename = "catalogNum", skip_serializing_if = "String::is_empty")]
    catalog_num: String,
    comment: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    bpm: Option<i32>,
    lyrics: String,
    #[serde(rename = "explicitStatus", skip_serializing_if = "String::is_empty")]
    explicit_status: String,
    #[serde(rename = "originalYear")]
    original_year: i32,
    #[serde(rename = "originalDate", skip_serializing_if = "String::is_empty")]
    original_date: String,
    #[serde(rename = "releaseYear")]
    release_year: i32,
    #[serde(rename = "releaseDate", skip_serializing_if = "String::is_empty")]
    release_date: String,
    year: i32,
    date: String,
    #[serde(rename = "mbzRecordingID", skip_serializing_if = "String::is_empty")]
    mbz_recording_id: String,
    #[serde(rename = "mbzReleaseTrackID", skip_serializing_if = "String::is_empty")]
    mbz_release_track_id: String,
    #[serde(rename = "mbzAlbumID", skip_serializing_if = "String::is_empty")]
    mbz_album_id: String,
    #[serde(rename = "mbzReleaseGroupID", skip_serializing_if = "String::is_empty")]
    mbz_release_group_id: String,
    #[serde(rename = "mbzAlbumType", skip_serializing_if = "String::is_empty")]
    mbz_album_type: String,
    #[serde(rename = "rgAlbumPeak", skip_serializing_if = "Option::is_none")]
    rg_album_peak: Option<f64>,
    #[serde(rename = "rgAlbumGain", skip_serializing_if = "Option::is_none")]
    rg_album_gain: Option<f64>,
    #[serde(rename = "rgTrackPeak", skip_serializing_if = "Option::is_none")]
    rg_track_peak: Option<f64>,
    #[serde(rename = "rgTrackGain", skip_serializing_if = "Option::is_none")]
    rg_track_gain: Option<f64>,
    #[serde(skip_serializing_if = "HashMap::is_empty")]
    participants: HashMap<String, Vec<ScanArtist>>,
    /// Album-level tags persisted on media files (mirrors Go TagMainMappings album:true).
    #[serde(skip_serializing_if = "HashMap::is_empty")]
    tags: HashMap<String, Vec<String>>,
    #[serde(rename = "pid", skip_serializing_if = "String::is_empty")]
    pid: String,
    #[serde(rename = "albumId", skip_serializing_if = "String::is_empty")]
    album_id: String,
    /// Precomputed FTS secondary tokens so Go persistence skips a second normalize RPC.
    #[serde(rename = "searchNormalized", skip_serializing_if = "String::is_empty")]
    search_normalized: String,
}

pub fn map_to_json(tags: &HashMap<String, Vec<String>>, path: &Path, lyrics_json: Option<&str>) -> Option<String> {
    map_to_json_with_pid(tags, path, lyrics_json, None, 0, "", None)
}

pub fn map_to_json_with_pid(
    tags: &HashMap<String, Vec<String>>,
    path: &Path,
    lyrics_json: Option<&str>,
    pid_config: Option<&crate::compute_pid::PidConfig>,
    library_id: i32,
    file_path: &str,
    map_config: Option<&MapMediaConfig>,
) -> Option<String> {
    let config = map_config.cloned().unwrap_or_default();
    let mut mapped = map_tags(tags, path, lyrics_json.unwrap_or("[]"), &config)?;
    if let Some(config) = pid_config {
        let input = crate::compute_pid::PidInput {
            library_id,
            path: file_path,
            title: &mapped.title,
            album: &mapped.album,
            album_artist: &mapped.album_artist,
            track_number: mapped.track_number,
            disc_number: mapped.disc_number,
            tags,
            mbz_recording_id: &mapped.mbz_recording_id,
            mbz_release_track_id: &mapped.mbz_release_track_id,
            mbz_album_id: &mapped.mbz_album_id,
            mbz_album_comment: &mapped.comment,
            release_date: &mapped.release_date,
        };
        let pids = crate::compute_pid::compute_pids(&input, config);
        mapped.pid = pids.track_pid;
        mapped.album_id = pids.album_id;
    }
    serde_json::to_string(&mapped).ok()
}

fn map_tags(
    tags: &HashMap<String, Vec<String>>,
    path: &Path,
    lyrics_json: &str,
    config: &MapMediaConfig,
) -> Option<ScanMediaFile> {
    let title = first_ref(tags, "title");
    let album = first_ref(tags, "album");
    if title.is_empty() && album.is_empty() {
        return None;
    }

    let artist = display_artist(tags, config);
    let album_artist = display_album_artist(tags, config);
    let (original_date, release_date, date) = map_dates(tags);
    let (track_number, _) = track_tuple(tags);
    let (disc_number, _) = disc_tuple(tags);
    let explicit = map_explicit(non_empty_or_ref(
        first_ref(tags, "itunesadvisory"),
        first_ref(tags, "explicit"),
    ));

    let mut mapped = ScanMediaFile {
        title: if title.is_empty() {
            path.file_stem()?.to_str()?.to_owned()
        } else {
            title.to_owned()
        },
        album: album.to_owned(),
        artist: artist.clone(),
        album_artist: album_artist.clone(),
        sort_title: first(tags, "titlesort"),
        sort_album_name: first(tags, "albumsort"),
        sort_artist_name: non_empty_or(first(tags, "artistsort"), first(tags, "albumartistsort")),
        sort_album_artist_name: first(tags, "albumartistsort"),
        order_title: sanitize_sort(title),
        order_album_name: sanitize_sort_no_article(album),
        compilation: parse_bool(first_ref(tags, "compilation")),
        track_number,
        disc_number,
        disc_subtitle: first(tags, "discsubtitle"),
        catalog_num: first(tags, "catalognumber"),
        comment: first(tags, "comment"),
        bpm: parse_bpm(first_ref(tags, "bpm")),
        lyrics: lyrics_json.to_owned(),
        explicit_status: explicit,
        original_year: year_from_date(&original_date),
        original_date,
        release_year: year_from_date(&release_date),
        release_date,
        year: year_from_date(&date),
        date,
        mbz_recording_id: mbz_recording_id(tags),
        mbz_release_track_id: mbz_release_track_id(tags),
        mbz_album_id: first(tags, "musicbrainz_albumid"),
        mbz_release_group_id: first(tags, "musicbrainz_releasegroupid"),
        mbz_album_type: first(tags, "releasetype"),
        rg_album_peak: parse_float(first_ref(tags, "replaygain_album_peak")),
        rg_album_gain: map_gain(
            first_ref(tags, "replaygain_album_gain"),
            first_ref(tags, "r128_album_gain"),
        ),
        rg_track_peak: parse_float(first_ref(tags, "replaygain_track_peak")),
        rg_track_gain: map_gain(
            first_ref(tags, "replaygain_track_gain"),
            first_ref(tags, "r128_track_gain"),
        ),
        participants: map_participants(tags, &artist, &album_artist, config),
        tags: map_album_tags(tags),
        pid: String::new(),
        album_id: String::new(),
        search_normalized: String::new(),
    };
    // Match Go PostMapArgs: title/album/artist/album_artist (FullTitle ≈ title unless Subsonic.AppendSubtitle).
    let normalized = fts_normalize::normalize_for_fts(&[
        mapped.title.clone(),
        mapped.album.clone(),
        mapped.artist.clone(),
        mapped.album_artist.clone(),
    ]);
    mapped.search_normalized = normalized;
    Some(mapped)
}

/// Extracts album-level tags that Go would keep after `clean()` + `TagMainMappings` filtering.
fn map_album_tags(tags: &HashMap<String, Vec<String>>) -> HashMap<String, Vec<String>> {
    let mut out = HashMap::new();
    insert_album_tag(
        &mut out,
        "albumversion",
        collect_tag_values(tags, &["albumversion", "musicbrainz_albumcomment"]),
    );
    insert_album_tag(&mut out, "genre", split_tag_values(tags, "genre", &[";", "/", ","]));
    insert_album_tag(&mut out, "mood", split_tag_values(tags, "mood", &[";", "/", ","]));
    insert_album_tag(
        &mut out,
        "tracktotal",
        collect_tag_values(tags, &["tracktotal", "totaltracks"]),
    );
    insert_album_tag(
        &mut out,
        "disctotal",
        collect_tag_values(tags, &["disctotal", "totaldiscs"]),
    );
    insert_album_tag(
        &mut out,
        "releasetype",
        split_tag_values(tags, "releasetype", &[","]),
    );
    out
}

fn insert_album_tag(out: &mut HashMap<String, Vec<String>>, key: &str, values: Vec<String>) {
    if !values.is_empty() {
        out.insert(key.to_owned(), values);
    }
}

fn collect_tag_values(tags: &HashMap<String, Vec<String>>, keys: &[&str]) -> Vec<String> {
    let mut seen = HashSet::new();
    let mut values = Vec::new();
    for key in keys {
        if let Some(raw) = tags.get(*key) {
            for value in raw {
                let value = value.trim();
                if value.is_empty() || !seen.insert(value.to_owned()) {
                    continue;
                }
                values.push(value.to_owned());
            }
        }
    }
    values
}

fn split_tag_values(
    tags: &HashMap<String, Vec<String>>,
    key: &str,
    separators: &[&str],
) -> Vec<String> {
    let mut seen = HashSet::new();
    let mut values = Vec::new();
    let Some(raw) = tags.get(key) else {
        return values;
    };
    for value in raw {
        let mut parts = vec![value.as_str()];
        for separator in separators {
            let mut next = Vec::new();
            for part in parts {
                next.extend(part.split(separator));
            }
            parts = next;
        }
        for part in parts {
            let part = part.trim();
            if part.is_empty() || !seen.insert(part.to_owned()) {
                continue;
            }
            values.push(part.to_owned());
        }
    }
    values
}

fn map_participants(
    tags: &HashMap<String, Vec<String>>,
    display_artist: &str,
    display_album_artist: &str,
    config: &MapMediaConfig,
) -> HashMap<String, Vec<ScanArtist>> {
    let mut out = HashMap::new();
    out.insert(
        "artist".to_owned(),
        artists_from_tags(
            tags,
            "artist",
            "artists",
            "artistsort",
            "musicbrainz_artistid",
            &config.artists_split,
            &config.artist_split_exceptions,
        ),
    );
    let mut album_artists = artists_from_tags(
        tags,
        "albumartist",
        "albumartists",
        "albumartistsort",
        "musicbrainz_albumartistid",
        &config.artists_split,
        &config.artist_split_exceptions,
    );
    if album_artists.is_empty() {
        album_artists.push(fallback_album_artist(
            display_album_artist,
            out.get("artist").map(Vec::as_slice).unwrap_or(&[]),
        ));
    }
    out.insert("albumartist".to_owned(), album_artists);

    for (role, key, sort_key, mbid_key) in [
        ("composer", "composer", "composersort", "musicbrainz_composerid"),
        ("conductor", "conductor", "conductorsort", "musicbrainz_conductorid"),
        ("lyricist", "lyricist", "lyricistsort", "musicbrainz_lyricistid"),
        ("arranger", "arranger", "arrangersort", "musicbrainz_arrangerid"),
        ("producer", "producer", "producersort", "musicbrainz_producerid"),
        ("director", "director", "directorsort", "musicbrainz_directorid"),
        ("engineer", "engineer", "engineersort", "musicbrainz_engineerid"),
        ("mixer", "mixer", "mixersort", "musicbrainz_mixerid"),
        ("remixer", "remixer", "remixersort", "musicbrainz_remixerid"),
        ("djmixer", "djmixer", "djmixersort", "musicbrainz_djmixerid"),
    ] {
        let artists = artists_from_tags(
            tags,
            key,
            key,
            sort_key,
            mbid_key,
            &config.roles_split,
            &config.artist_split_exceptions,
        );
        if !artists.is_empty() {
            out.insert(role.to_owned(), artists);
        }
    }

    if out["artist"].is_empty() && !display_artist.is_empty() {
        out.insert("artist".to_owned(), vec![scan_artist(display_artist, "", "")]);
    }
    out
}

fn artists_from_tags(
    tags: &HashMap<String, Vec<String>>,
    single: &str,
    plural: &str,
    sort_key: &str,
    mbid_key: &str,
    split: &[String],
    exceptions: &[String],
) -> Vec<ScanArtist> {
    let names = get_participant_values(tags, single, plural, split, exceptions);
    if names.is_empty() {
        return Vec::new();
    }
    let sorts = tag_values(tags, sort_key);
    let mbids = tag_values(tags, mbid_key);
    names
        .into_iter()
        .enumerate()
        .map(|(idx, name)| {
            scan_artist(
                &name,
                sorts
                    .and_then(|values| values.get(idx))
                    .map(String::as_str)
                    .unwrap_or_default(),
                mbids
                    .and_then(|values| values.get(idx))
                    .map(String::as_str)
                    .unwrap_or_default(),
            )
        })
        .fold(Vec::new(), |mut artists, artist| {
            if artists.iter().any(|existing| existing.name == artist.name) {
                return artists;
            }
            artists.push(artist);
            artists
        })
}

/// Mirrors Go getArtistValues / getRoleValues participant splitting.
fn get_participant_values(
    tags: &HashMap<String, Vec<String>>,
    single: &str,
    plural: &str,
    split: &[String],
    exceptions: &[String],
) -> Vec<String> {
    let multi: Vec<String> = filtered_value_refs(tags, plural)
        .into_iter()
        .map(str::to_owned)
        .collect();
    if !multi.is_empty() {
        return multi;
    }
    let single_vals: Vec<String> = filtered_value_refs(tags, single)
        .into_iter()
        .map(str::to_owned)
        .collect();
    if single_vals.len() != 1 {
        return single_vals;
    }
    tag_clean::split_participant_values(&single_vals, split, exceptions)
}

fn scan_artist(name: &str, sort: &str, mbid: &str) -> ScanArtist {
    ScanArtist {
        name: name.to_owned(),
        sort_artist_name: sort.to_owned(),
        order_artist_name: sanitize_sort_no_article(name),
        mbz_artist_id: mbid.to_owned(),
        sub_role: String::new(),
    }
}

fn fallback_album_artist(display_album_artist: &str, artist_role: &[ScanArtist]) -> ScanArtist {
    for artist in artist_role {
        if artist.name == display_album_artist {
            return ScanArtist {
                name: display_album_artist.to_owned(),
                sort_artist_name: artist.sort_artist_name.clone(),
                order_artist_name: artist.order_artist_name.clone(),
                mbz_artist_id: artist.mbz_artist_id.clone(),
                sub_role: String::new(),
            };
        }
    }
    scan_artist(display_album_artist, "", "")
}

fn display_artist(tags: &HashMap<String, Vec<String>>, config: &MapMediaConfig) -> String {
    let values = unique_non_empty_refs(filtered_value_refs(tags, "artists"));
    let values = if values.is_empty() {
        unique_non_empty_refs(filtered_value_refs(tags, "artist"))
    } else {
        values
    };
    join_artists(&values, &config.artist_joiner)
}

fn display_album_artist(tags: &HashMap<String, Vec<String>>, config: &MapMediaConfig) -> String {
    let values = unique_non_empty_refs(filtered_value_refs(tags, "albumartists"));
    let values = if values.is_empty() {
        unique_non_empty_refs(filtered_value_refs(tags, "albumartist"))
    } else {
        values
    };
    let joined = join_artists(&values, &config.artist_joiner);
    if !joined.is_empty() {
        return joined;
    }
    display_artist(tags, config)
}

fn join_artists(values: &[&str], joiner: &str) -> String {
    values.join(joiner)
}

fn unique_non_empty_refs(values: Vec<&str>) -> Vec<&str> {
    let mut seen = HashSet::new();
    let mut out = Vec::new();
    for value in values {
        if value.is_empty() || !seen.insert(value) {
            continue;
        }
        out.push(value);
    }
    out
}

fn filtered_value_refs<'a>(tags: &'a HashMap<String, Vec<String>>, key: &str) -> Vec<&'a str> {
    tags.get(key)
        .map(|values| {
            values
                .iter()
                .map(String::as_str)
                .filter(|value| !value.is_empty())
                .collect()
        })
        .unwrap_or_default()
}

fn tag_values<'a>(
    tags: &'a HashMap<String, Vec<String>>,
    key: &str,
) -> Option<&'a [String]> {
    let values = tags.get(key)?;
    if values.iter().any(|value| !value.is_empty()) {
        Some(values.as_slice())
    } else {
        None
    }
}

fn first_ref<'a>(tags: &'a HashMap<String, Vec<String>>, key: &str) -> &'a str {
    tags.get(key)
        .and_then(|values| values.first())
        .map(String::as_str)
        .unwrap_or("")
}

fn mbz_recording_id(tags: &HashMap<String, Vec<String>>) -> String {
    let recording = first_ref(tags, "musicbrainz_recordingid");
    if !recording.is_empty() {
        return recording.to_owned();
    }
    if !first_ref(tags, "musicbrainz_releasetrackid").is_empty() {
        return first(tags, "musicbrainz_trackid");
    }
    String::new()
}

fn mbz_release_track_id(tags: &HashMap<String, Vec<String>>) -> String {
    non_empty_or_ref(
        first_ref(tags, "musicbrainz_releasetrackid"),
        first_ref(tags, "musicbrainz_trackid"),
    )
    .to_owned()
}

fn non_empty_or_ref<'a>(a: &'a str, b: &'a str) -> &'a str {
    if a.is_empty() { b } else { a }
}

fn first(tags: &HashMap<String, Vec<String>>, key: &str) -> String {
    first_ref(tags, key).to_owned()
}

fn parse_bool(value: &str) -> bool {
    matches!(value.trim().to_ascii_lowercase().as_str(), "1" | "true" | "yes")
}

fn parse_bpm(value: &str) -> Option<i32> {
    value.trim().parse::<f64>().ok().map(|v| v.round() as i32).filter(|&v| v != 0)
}

fn parse_float(value: &str) -> Option<f64> {
    let value = value.trim();
    if value.is_empty() {
        return None;
    }
    if !value
        .chars()
        .all(|c| c.is_ascii_digit() || matches!(c, '.' | '-' | '+' | 'e' | 'E'))
    {
        return None;
    }
    value.parse().ok()
}

fn strip_db(value: &str) -> String {
    value.replace("dB", "").replace("db", "").trim().to_owned()
}

fn map_gain(rg: &str, r128: &str) -> Option<f64> {
    let stripped = strip_db(rg);
    if let Some(v) = parse_float(&stripped) {
        return Some(v);
    }
    if let Ok(v) = r128.trim().parse::<i64>() {
        return Some(v as f64 / 256.0 + 5.0);
    }
    None
}

fn map_explicit(value: &str) -> String {
    match value.trim() {
        "1" | "4" => "e".to_owned(),
        "2" => "c".to_owned(),
        _ => String::new(),
    }
}

fn tuple(tags: &HashMap<String, Vec<String>>, key: &str, total_key: &str) -> (i32, i32) {
    let raw = first_ref(tags, key);
    if raw.is_empty() {
        return (0, 0);
    }
    let mut parts = raw.split('/');
    let first_num = parts.next().unwrap_or_default().parse().unwrap_or(0);
    let second = parts
        .next()
        .and_then(|v| v.parse().ok())
        .or_else(|| first_ref(tags, total_key).parse().ok())
        .unwrap_or(0);
    (first_num, second)
}

fn track_tuple(tags: &HashMap<String, Vec<String>>) -> (i32, i32) {
    let (number, total) = tuple(tags, "tracknumber", "tracktotal");
    if number != 0 {
        return (number, total);
    }
    tuple(tags, "track", "tracktotal")
}

fn disc_tuple(tags: &HashMap<String, Vec<String>>) -> (i32, i32) {
    let (number, total) = tuple(tags, "discnumber", "disctotal");
    if number != 0 {
        return (number, total);
    }
    tuple(tags, "disc", "disctotal")
}

fn map_dates(tags: &HashMap<String, Vec<String>>) -> (String, String, String) {
    let mut original = first(tags, "originaldate");
    let mut release = first(tags, "releasedate");
    let mut date = non_empty_or(first(tags, "date"), first(tags, "recordingdate"));
    if original.is_empty() {
        original = first(tags, "originalyear");
    }
    if release.is_empty() {
        release = first(tags, "releaseyear");
    }
    if date.is_empty() {
        date = first(tags, "year");
    }
    if !original.is_empty() && release.is_empty() && !date.is_empty() && date >= original {
        return (original.clone(), date.clone(), original);
    }
    let resolved = if !date.is_empty() {
        date.clone()
    } else if !original.is_empty() {
        original.clone()
    } else {
        release.clone()
    };
    (original, release, resolved)
}

fn non_empty_or(a: String, b: String) -> String {
    if a.is_empty() { b } else { a }
}

fn year_from_date(value: &str) -> i32 {
    value.get(0..4).and_then(|y| y.parse().ok()).unwrap_or(0)
}

fn sanitize_sort(value: &str) -> String {
    value.trim().to_owned()
}

fn sanitize_sort_no_article(value: &str) -> String {
    let lower = value.trim().to_ascii_lowercase();
    for prefix in ["the ", "a ", "an "] {
        if lower.starts_with(prefix) {
            return value[prefix.len()..].trim().to_owned();
        }
    }
    value.trim().to_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn splits_feat_artists_into_participants() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert(
            "artist".to_owned(),
            vec!["Artist Name feat. Someone Else".to_owned()],
        );
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""artist":"Artist Name feat. Someone Else""#));
        assert!(json.contains(r#""name":"Artist Name""#));
        assert!(json.contains(r#""name":"Someone Else""#));
    }

    #[test]
    fn respects_artist_split_exceptions() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert(
            "artist".to_owned(),
            vec!["Someone feat. Else".to_owned()],
        );
        let config = MapMediaConfig {
            artist_split_exceptions: vec!["Someone feat. Else".to_owned()],
            ..MapMediaConfig::with_defaults()
        };
        let json = map_to_json_with_pid(
            &tags,
            Path::new("music/song.mp3"),
            Some("[]"),
            None,
            0,
            "",
            Some(&config),
        )
        .expect("json");
        assert!(json.contains(r#""name":"Someone feat. Else""#));
        assert!(!json.contains(r#""name":"Someone""#));
    }

    #[test]
    fn maps_basic_tags() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("artist".to_owned(), vec!["Artist".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains("Song"));
        assert!(json.contains("Album"));
    }

    #[test]
    fn maps_legacy_release_date_mapping() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("date".to_owned(), vec!["2020-05-15".to_owned()]);
        tags.insert("originaldate".to_owned(), vec!["2019-02-10".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""releaseDate":"2020-05-15""#));
    }

    #[test]
    fn deduplicates_repeated_artist_values() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert(
            "artist".to_owned(),
            vec!["Taylor Swift".to_owned(), "Taylor Swift".to_owned()],
        );
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""artist":"Taylor Swift""#));
        assert!(!json.contains("Taylor Swift; Taylor Swift"));
        assert!(!json.contains("Taylor Swift • Taylor Swift"));
    }

    #[test]
    fn prefers_plural_artist_tags_without_duplicating_singular() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("artist".to_owned(), vec!["Taylor Swift".to_owned()]);
        tags.insert("artists".to_owned(), vec!["Taylor Swift".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""artist":"Taylor Swift""#));
    }

    #[test]
    fn joins_multiple_artists_with_nav_joiner() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert(
            "artists".to_owned(),
            vec!["Artist A".to_owned(), "Artist B".to_owned()],
        );
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""artist":"Artist A • Artist B""#));
    }

    #[test]
    fn inherits_artist_mbid_for_albumartist_fallback() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Help!".to_owned()]);
        tags.insert("album".to_owned(), vec!["Help!".to_owned()]);
        tags.insert("artist".to_owned(), vec!["The Beatles".to_owned()]);
        tags.insert("artistsort".to_owned(), vec!["Beatles, The".to_owned()]);
        tags.insert(
            "musicbrainz_artistid".to_owned(),
            vec!["18220d3d-16d4-402f-95b3-cd08acb043f1".to_owned()],
        );
        let json = map_to_json(&tags, Path::new("music/help.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""mbzArtistId":"18220d3d-16d4-402f-95b3-cd08acb043f1""#));
        assert!(json.contains(r#""sortArtistName":"Beatles, The""#));
        assert!(json.contains(r#""albumartist""#));
    }

    #[test]
    fn maps_album_tags_for_persistence() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("genre".to_owned(), vec!["Rock; Pop".to_owned()]);
        tags.insert("mood".to_owned(), vec!["Happy".to_owned()]);
        tags.insert("tracktotal".to_owned(), vec!["12".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""genre":["Rock","Pop"]"#), "json={json}");
        assert!(json.contains(r#""mood":["Happy"]"#));
        assert!(json.contains(r#""tracktotal":["12"]"#));
    }

    #[test]
    fn maps_picard_track_and_disc_aliases() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("track".to_owned(), vec!["3/12".to_owned()]);
        tags.insert("disc".to_owned(), vec!["2/2".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""trackNumber":3"#), "json={json}");
        assert!(json.contains(r#""discNumber":2"#), "json={json}");
    }

    #[test]
    fn maps_musicbrainz_release_track_aliases() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert(
            "musicbrainz_releasetrackid".to_owned(),
            vec!["release-track-id".to_owned()],
        );
        tags.insert(
            "musicbrainz_trackid".to_owned(),
            vec!["recording-id".to_owned()],
        );
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""mbzReleaseTrackID":"release-track-id""#), "json={json}");
        assert!(json.contains(r#""mbzRecordingID":"recording-id""#), "json={json}");
    }

    #[test]
    fn maps_properly_tagged_dates() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Song".to_owned()]);
        tags.insert("album".to_owned(), vec!["Album".to_owned()]);
        tags.insert("originaldate".to_owned(), vec!["1978-09-10".to_owned()]);
        tags.insert("date".to_owned(), vec!["1977-03-04".to_owned()]);
        tags.insert("releasedate".to_owned(), vec!["2002-01-02".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/song.mp3"), Some("[]")).expect("json");
        assert!(json.contains(r#""date":"1977-03-04""#));
        assert!(json.contains(r#""originalDate":"1978-09-10""#));
        assert!(json.contains(r#""releaseDate":"2002-01-02""#));
    }

    #[test]
    fn embeds_search_normalized_for_punctuated_artists() {
        let mut tags = HashMap::new();
        tags.insert("title".to_owned(), vec!["Losing My Religion".to_owned()]);
        tags.insert("album".to_owned(), vec!["Out of Time".to_owned()]);
        tags.insert("artist".to_owned(), vec!["R.E.M.".to_owned()]);
        tags.insert("albumartist".to_owned(), vec!["R.E.M.".to_owned()]);
        let json = map_to_json(&tags, Path::new("music/losing.mp3"), Some("[]")).expect("json");
        assert!(
            json.contains(r#""searchNormalized":"REM""#),
            "expected embedded searchNormalized, json={json}"
        );
    }
}
