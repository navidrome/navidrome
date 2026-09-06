pub mod proto {
    tonic::include_proto!("navidrome.search.v1");
}

pub mod grpc;

use std::collections::HashSet;
use std::io::{self, BufRead, BufReader, BufWriter, Write};
use std::path::Path;
use std::sync::RwLock;

use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};
use tantivy::collector::TopDocs;
use tantivy::query::{BooleanQuery, BoostQuery, FuzzyTermQuery, Occur, Query, TermQuery};
use tantivy::schema::{
    Field, INDEXED, IndexRecordOption, STORED, STRING, Schema, TantivyDocument, TextFieldIndexing,
    TextOptions, Value,
};
use tantivy::tokenizer::{LowerCaser, NgramTokenizer, TextAnalyzer};
use tantivy::{Index, IndexReader, IndexWriter, ReloadPolicy, Term, doc};

const PROTOCOL_VERSION: u32 = 1;
const MAX_DOCUMENTS_PER_REQUEST: usize = 250_000;
const MAX_RESULTS: usize = 500;
const MAX_SEARCH_GROUPS: usize = 8;
const MAX_QUERY_CHARS: usize = 256;
const WRITER_MEMORY_BYTES: usize = 32 * 1024 * 1024;
const MAX_NGRAM_CONJUNCTS: usize = 12;
const NGRAM_TOKENIZER: &str = "navidrome_ngram";

#[derive(Debug, Clone, Deserialize)]
struct SearchDocument {
    key: String,
    id: String,
    kind: String,
    #[serde(default)]
    library_ids: Vec<u64>,
    primary: String,
    #[serde(default)]
    secondary: String,
}

#[derive(Debug, Deserialize)]
struct SearchSpec {
    kind: String,
    #[serde(default)]
    offset: usize,
    #[serde(default)]
    limit: usize,
}

#[derive(Debug, Deserialize)]
#[serde(tag = "op", rename_all = "snake_case")]
enum Request {
    BeginReplace,
    Append {
        documents: Vec<SearchDocument>,
    },
    CommitReplace,
    AbortReplace,
    Upsert {
        documents: Vec<SearchDocument>,
    },
    Delete {
        keys: Vec<String>,
    },
    Commit,
    SearchAll {
        query: String,
        #[serde(default)]
        library_ids: Vec<u64>,
        searches: Vec<SearchSpec>,
    },
    NormalizeFts {
        values: Vec<String>,
    },
}

#[derive(Debug, Serialize)]
struct Hit {
    id: String,
    score: f32,
}

#[derive(Debug, Serialize)]
struct SearchGroup {
    kind: String,
    hits: Vec<Hit>,
}

#[derive(Debug, Serialize)]
struct Response {
    protocol: u32,
    ok: bool,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    groups: Vec<SearchGroup>,
    indexed: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    normalized: Option<String>,
}

#[derive(Clone, Copy)]
struct Fields {
    key: Field,
    id: Field,
    kind: Field,
    library_id: Field,
    exact: Field,
    primary: Field,
    secondary: Field,
}

struct Engine {
    index: Index,
    reader: IndexReader,
    writer: IndexWriter,
    fields: Fields,
    indexed: u64,
    replace_in_progress: bool,
}

impl Engine {
    fn new() -> Result<Self> {
        let mut schema = Schema::builder();
        let key = schema.add_text_field("key", STRING);
        let id = schema.add_text_field("id", STORED);
        let kind = schema.add_text_field("kind", STRING);
        let library_id = schema.add_u64_field("library_id", INDEXED);
        let exact = schema.add_text_field("exact", STRING);
        let ngram_options = TextOptions::default().set_indexing_options(
            TextFieldIndexing::default()
                .set_tokenizer(NGRAM_TOKENIZER)
                .set_index_option(IndexRecordOption::WithFreqs),
        );
        let primary = schema.add_text_field("primary", ngram_options.clone());
        let secondary = schema.add_text_field("secondary", ngram_options);
        let schema = schema.build();

        let index = open_index(&schema)?;
        let analyzer = TextAnalyzer::builder(NgramTokenizer::new(2, 3, false)?)
            .filter(LowerCaser)
            .build();
        index.tokenizers().register(NGRAM_TOKENIZER, analyzer);
        let writer = index.writer(WRITER_MEMORY_BYTES)?;
        let reader = index
            .reader_builder()
            .reload_policy(ReloadPolicy::Manual)
            .try_into()?;

        Ok(Self {
            index,
            reader,
            writer,
            fields: Fields {
                key,
                id,
                kind,
                library_id,
                exact,
                primary,
                secondary,
            },
            indexed: 0,
            replace_in_progress: false,
        })
    }

    fn replace(&mut self, documents: Vec<SearchDocument>) -> Result<()> {
        validate_document_batch(&documents)?;
        self.begin_replace()?;
        if let Err(error) = self.append(documents) {
            self.abort_replace()?;
            return Err(error);
        }
        self.commit_replace()
    }

    fn begin_replace(&mut self) -> Result<()> {
        if self.replace_in_progress {
            bail!("search index replacement is already in progress");
        }
        self.writer.delete_all_documents()?;
        self.replace_in_progress = true;
        Ok(())
    }

    fn append(&mut self, documents: Vec<SearchDocument>) -> Result<()> {
        if !self.replace_in_progress {
            bail!("begin_replace is required before append");
        }
        validate_document_batch(&documents)?;
        self.add_documents(documents)?;
        Ok(())
    }

    fn commit_replace(&mut self) -> Result<()> {
        if !self.replace_in_progress {
            bail!("no search index replacement is in progress");
        }
        self.commit()?;
        self.replace_in_progress = false;
        Ok(())
    }

    fn abort_replace(&mut self) -> Result<()> {
        if !self.replace_in_progress {
            bail!("no search index replacement is in progress");
        }
        self.writer.rollback()?;
        self.replace_in_progress = false;
        self.indexed = self.reader.searcher().num_docs();
        Ok(())
    }

    fn upsert(&mut self, documents: Vec<SearchDocument>) -> Result<()> {
        if self.replace_in_progress {
            bail!("cannot upsert while replacing the search index");
        }
        validate_document_batch(&documents)?;
        for document in &documents {
            self.writer
                .delete_term(Term::from_field_text(self.fields.key, &document.key));
        }
        self.add_documents(documents)?;
        Ok(())
    }

    fn delete(&mut self, keys: Vec<String>) -> Result<()> {
        if self.replace_in_progress {
            bail!("cannot delete while replacing the search index");
        }
        if keys.len() > MAX_DOCUMENTS_PER_REQUEST {
            bail!("delete contains too many keys");
        }
        let mut unique = HashSet::with_capacity(keys.len());
        for key in &keys {
            if key.is_empty() {
                bail!("delete keys must be non-empty");
            }
            if !unique.insert(key.as_str()) {
                continue;
            }
            self.writer
                .delete_term(Term::from_field_text(self.fields.key, key));
        }
        Ok(())
    }

    fn add_documents(&mut self, documents: Vec<SearchDocument>) -> Result<()> {
        for document in documents {
            let secondary = enrich_secondary_with_fts(&document.primary, &document.secondary);
            let normalized_primary = normalize(&document.primary);
            let normalized_secondary = if secondary.is_empty() {
                String::new()
            } else {
                normalize(&secondary)
            };
            let mut indexed = tantivy::doc!(
                self.fields.key => document.key.as_str(),
                self.fields.id => document.id.as_str(),
                self.fields.kind => document.kind.as_str(),
                self.fields.exact => normalized_primary.as_str(),
                self.fields.primary => normalized_primary.as_str(),
                self.fields.secondary => normalized_secondary.as_str(),
            );
            for library_id in document.library_ids {
                indexed.add_u64(self.fields.library_id, library_id);
            }
            self.writer.add_document(indexed)?;
        }
        Ok(())
    }

    fn commit(&mut self) -> Result<()> {
        self.writer.commit()?;
        self.reader.reload()?;
        self.indexed = self.reader.searcher().num_docs();
        Ok(())
    }

    fn search(
        &self,
        query: &str,
        kind: &str,
        library_ids: &[u64],
        offset: usize,
        limit: usize,
    ) -> Result<Vec<Hit>> {
        let normalized = normalize(query);
        if !is_searchable_query(query) || limit == 0 {
            return Ok(Vec::new());
        }
        let limit = limit.min(MAX_RESULTS);
        let requested = offset.saturating_add(limit).min(MAX_RESULTS);

        let exact_query: Box<dyn Query> = Box::new(BoostQuery::new(
            Box::new(TermQuery::new(
                Term::from_field_text(self.fields.exact, &normalized),
                IndexRecordOption::Basic,
            )),
            12.0,
        ));
        let text_query = match self.ngram_query(&normalized) {
            Ok(query) => query,
            Err(_) if normalized.chars().any(is_cjk_char) => {
                Box::new(TermQuery::new(
                    Term::from_field_text(self.fields.primary, &normalized),
                    IndexRecordOption::WithFreqs,
                )) as Box<dyn Query>
            }
            Err(error) => return Err(error),
        };
        let mut relevance_clauses = vec![(Occur::Should, exact_query), (Occur::Should, text_query)];
        // N-grams provide fast substring matching, while a bounded fuzzy prefix
        // catches common Latin-script typos and transpositions. Keep it off the
        // much larger Unicode term space and below the exact-match boost.
        if normalized.is_ascii() && normalized.len() >= 3 {
            relevance_clauses.push((
                Occur::Should,
                Box::new(BoostQuery::new(
                    Box::new(FuzzyTermQuery::new_prefix(
                        Term::from_field_text(self.fields.exact, &normalized),
                        1,
                        true,
                    )),
                    7.0,
                )),
            ));
        }
        let relevance: Box<dyn Query> = Box::new(BooleanQuery::new(relevance_clauses));

        let mut clauses: Vec<(Occur, Box<dyn Query>)> = vec![
            (
                Occur::Must,
                Box::new(TermQuery::new(
                    Term::from_field_text(self.fields.kind, kind),
                    IndexRecordOption::Basic,
                )),
            ),
            (Occur::Must, relevance),
        ];
        if !library_ids.is_empty() {
            let scopes = library_ids
                .iter()
                .map(|library_id| {
                    (
                        Occur::Should,
                        Box::new(TermQuery::new(
                            Term::from_field_u64(self.fields.library_id, *library_id),
                            IndexRecordOption::Basic,
                        )) as Box<dyn Query>,
                    )
                })
                .collect();
            clauses.push((Occur::Must, Box::new(BooleanQuery::new(scopes))));
        }

        let searcher = self.reader.searcher();
        let top_docs = searcher.search(
            &BooleanQuery::new(clauses),
            &TopDocs::with_limit(requested).order_by_score(),
        )?;
        let mut hits = Vec::with_capacity(limit);
        for (score, address) in top_docs.into_iter().skip(offset) {
            hits.push(Hit {
                id: load_hit_id(&searcher, self.fields.id, address)?,
                score,
            });
        }
        Ok(hits)
    }

    fn ngram_query(&self, query: &str) -> Result<Box<dyn Query>> {
        let mut analyzer = self
            .index
            .tokenizers()
            .get(NGRAM_TOKENIZER)
            .context("n-gram tokenizer is not registered")?;
        let mut stream = analyzer.token_stream(query);
        let mut seen = HashSet::with_capacity(query.len() / 2);
        let mut tokens = Vec::with_capacity(query.len() / 2);
        while stream.advance() {
            let text = stream.token().text.clone();
            if seen.insert(text.clone()) {
                tokens.push(text);
            }
        }
        if tokens.is_empty() {
            bail!("query produced no searchable n-grams");
        }
        if tokens.len() > MAX_NGRAM_CONJUNCTS {
            tokens.truncate(MAX_NGRAM_CONJUNCTS);
        }
        let mut terms = Vec::with_capacity(tokens.len());
        for text in tokens {
            terms.push((
                Occur::Must,
                Box::new(BooleanQuery::new(vec![
                    (
                        Occur::Should,
                        Box::new(BoostQuery::new(
                            Box::new(TermQuery::new(
                                Term::from_field_text(self.fields.primary, &text),
                                IndexRecordOption::WithFreqs,
                            )),
                            4.0,
                        )),
                    ),
                    (
                        Occur::Should,
                        Box::new(TermQuery::new(
                            Term::from_field_text(self.fields.secondary, &text),
                            IndexRecordOption::WithFreqs,
                        )),
                    ),
                ])) as Box<dyn Query>,
            ));
        }
        Ok(Box::new(BooleanQuery::new(terms)))
    }
}

fn load_hit_id(
    searcher: &tantivy::Searcher,
    id_field: Field,
    address: tantivy::DocAddress,
) -> Result<String> {
    let document: TantivyDocument = searcher.doc(address)?;
    document
        .get_first(id_field)
        .and_then(|value| value.as_str().map(str::to_owned))
        .context("search result is missing id")
}

fn validate_document_batch(documents: &[SearchDocument]) -> Result<()> {
    if documents.len() > MAX_DOCUMENTS_PER_REQUEST {
        bail!(
            "request contains {} documents; maximum is {MAX_DOCUMENTS_PER_REQUEST}",
            documents.len()
        );
    }
    let mut keys = HashSet::with_capacity(documents.len());
    for document in documents {
        if document.key.is_empty()
            || document.id.is_empty()
            || document.kind.is_empty()
            || document.primary.is_empty()
        {
            bail!("search documents require key, id, kind, and primary");
        }
        if !keys.insert(document.key.as_str()) {
            bail!("search document keys must be unique within a request");
        }
    }
    Ok(())
}

/// Fold fts-normalize variants into secondary before Tantivy n-gram indexing.
/// Go may still send DB search_normalized tokens; this covers legacy rows and
/// avoids a metadata NormalizeFtsBatch hop during Apply append/upsert.
fn enrich_secondary_with_fts(primary: &str, secondary: &str) -> String {
    let mut values = Vec::with_capacity(2);
    if !primary.is_empty() {
        values.push(primary.to_owned());
    }
    if !secondary.is_empty() {
        values.push(secondary.to_owned());
    }
    let fts_extra = fts_normalize::normalize_for_fts(&values);
    if fts_extra.is_empty() {
        return secondary.to_owned();
    }
    if secondary.is_empty() {
        return fts_extra;
    }
    let mut enriched = String::with_capacity(secondary.len() + 1 + fts_extra.len());
    enriched.push_str(secondary);
    enriched.push(' ');
    enriched.push_str(&fts_extra);
    enriched
}

fn normalize(value: &str) -> String {
    let mut normalized = String::with_capacity(value.len());
    let mut needs_space = false;
    for ch in value.chars().flat_map(char::to_lowercase) {
        if ch.is_alphanumeric() {
            if needs_space && !normalized.is_empty() {
                normalized.push(' ');
            }
            normalized.push(ch);
            needs_space = false;
        } else {
            needs_space = true;
        }
    }
    normalized
}

/// Mirrors Go `rustSearchableQuery`: any CJK rune, or at least two alphanumeric chars.
fn is_searchable_query(query: &str) -> bool {
    let mut searchable = 0usize;
    for ch in query.chars() {
        if is_cjk_char(ch) {
            return true;
        }
        if ch.is_alphanumeric() {
            searchable += 1;
            if searchable >= 2 {
                return true;
            }
        }
    }
    false
}

fn is_cjk_char(ch: char) -> bool {
    matches!(
        ch,
        '\u{1100}'..='\u{11FF}'
            | '\u{3040}'..='\u{309F}'
            | '\u{30A0}'..='\u{30FF}'
            | '\u{3130}'..='\u{318F}'
            | '\u{4E00}'..='\u{9FFF}'
            | '\u{AC00}'..='\u{D7AF}'
    )
}

pub fn run() -> Result<()> {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut output = BufWriter::with_capacity(64 * 1024, stdout.lock());
    let engine: RwLock<Option<Engine>> = RwLock::new(None);

    for line in BufReader::with_capacity(256 * 1024, stdin.lock()).lines() {
        let line = line.context("reading search request")?;
        if line.trim().is_empty() {
            continue;
        }
        let indexed = engine
            .read()
            .ok()
            .and_then(|guard| guard.as_ref().map(|engine| engine.indexed))
            .unwrap_or(0);
        let response = match serde_json::from_str::<Request>(&line) {
            Ok(request) => match handle_request_locked(&engine, request) {
                Ok(response) => response,
                Err(error) => Response {
                    protocol: PROTOCOL_VERSION,
                    ok: false,
                    groups: Vec::new(),
                    indexed: engine
                        .read()
                        .ok()
                        .and_then(|guard| guard.as_ref().map(|engine| engine.indexed))
                        .unwrap_or(indexed),
                    error: Some(format!("{error:#}")),
                    normalized: None,
                },
            },
            Err(error) => Response {
                protocol: PROTOCOL_VERSION,
                ok: false,
                groups: Vec::new(),
                indexed,
                error: Some(error.to_string()),
                normalized: None,
            },
        };
        serde_json::to_writer(&mut output, &response)?;
        output.write_all(b"\n")?;
        output.flush()?;
    }
    Ok(())
}

fn open_index(schema: &Schema) -> Result<Index> {
    if let Ok(path) = std::env::var("NAVIDROME_SEARCH_INDEX_PATH") {
        if !path.is_empty() {
            let path = Path::new(&path);
            std::fs::create_dir_all(path).with_context(|| format!("creating {}", path.display()))?;
            if path.join("meta.json").is_file() {
                return Index::open_in_dir(path).with_context(|| format!("opening {}", path.display()));
            }
            return Index::create_in_dir(path, schema.clone())
                .with_context(|| format!("creating {}", path.display()));
        }
    }
    Ok(Index::create_in_ram(schema.clone()))
}

fn handle_request_locked(engine: &RwLock<Option<Engine>>, request: Request) -> Result<Response> {
    let read_only = matches!(request, Request::SearchAll { .. } | Request::NormalizeFts { .. });
    if read_only {
        let guard = engine
            .read()
            .map_err(|error| anyhow::anyhow!("locking search engine for read: {error}"))?;
        let engine = guard
            .as_ref()
            .context("search engine is not initialized")?;
        return handle_read_request(engine, request);
    }
    let mut guard = engine
        .write()
        .map_err(|error| anyhow::anyhow!("locking search engine for write: {error}"))?;
    if guard.is_none() {
        *guard = Some(Engine::new()?);
    }
    let engine = guard.as_mut().expect("engine initialized above");
    handle_request(engine, request)
}

fn handle_read_request(engine: &Engine, request: Request) -> Result<Response> {
    let mut groups = Vec::new();
    match request {
        Request::SearchAll {
            query,
            library_ids,
            searches,
        } => {
            if query.chars().count() > MAX_QUERY_CHARS {
                bail!("search query exceeds {MAX_QUERY_CHARS} characters");
            }
            if searches.len() > MAX_SEARCH_GROUPS {
                bail!("search_all contains too many groups");
            }
            let mut kinds = HashSet::with_capacity(searches.len());
            for search in searches {
                if search.kind.is_empty() || !kinds.insert(search.kind.clone()) {
                    bail!("search_all groups require unique non-empty kinds");
                }
                groups.push(SearchGroup {
                    hits: engine.search(
                        &query,
                        &search.kind,
                        &library_ids,
                        search.offset,
                        search.limit,
                    )?,
                    kind: search.kind,
                });
            }
        }
        Request::NormalizeFts { values } => {
            return Ok(Response {
                protocol: PROTOCOL_VERSION,
                ok: true,
                groups: Vec::new(),
                indexed: engine.indexed,
                error: None,
                normalized: Some(fts_normalize::normalize_for_fts(&values)),
            });
        }
        _ => bail!("unsupported read-only search request"),
    }
    Ok(Response {
        protocol: PROTOCOL_VERSION,
        ok: true,
        groups,
        indexed: engine.indexed,
        error: None,
        normalized: None,
    })
}

fn handle_request(engine: &mut Engine, request: Request) -> Result<Response> {
    let groups = Vec::new();
    match request {
        Request::BeginReplace => {
            engine.begin_replace()?;
        }
        Request::Append { documents } => {
            engine.append(documents)?;
        }
        Request::CommitReplace => {
            engine.commit_replace()?;
        }
        Request::AbortReplace => {
            engine.abort_replace()?;
        }
        Request::Upsert { documents } => {
            engine.upsert(documents)?;
        }
        Request::Delete { keys } => {
            engine.delete(keys)?;
        }
        Request::Commit => {
            engine.commit()?;
        }
        Request::SearchAll { .. } | Request::NormalizeFts { .. } => {
            bail!("read-only search requests must use handle_read_request");
        }
    }
    Ok(Response {
        protocol: PROTOCOL_VERSION,
        ok: true,
        groups,
        indexed: engine.indexed,
        error: None,
        normalized: None,
    })
}

pub mod bench_support {
    use super::*;

    pub struct BenchEngine(Engine);

    impl BenchEngine {
        pub fn new() -> Result<Self> {
            Ok(Self(Engine::new()?))
        }

        pub fn load_documents(&mut self, count: usize) -> Result<()> {
            self.0.replace(bench_documents(count))
        }

        pub fn search_all(&self, query: &str, library_ids: &[u64]) -> Result<usize> {
            let mut total = 0usize;
            for kind in ["song", "album", "artist"] {
                total += self.0.search(query, kind, library_ids, 0, 10)?.len();
            }
            Ok(total)
        }
    }

    fn bench_documents(count: usize) -> Vec<SearchDocument> {
        (0..count)
            .map(|index| SearchDocument {
                key: format!("song:{index}"),
                id: format!("song-{index}"),
                kind: "song".to_owned(),
                library_ids: vec![1],
                primary: format!("Track {index} Blue Monday"),
                secondary: format!("Artist {index} R.E.M. Bjørk"),
            })
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn document(key: &str, id: &str, kind: &str, library_id: u64, primary: &str) -> SearchDocument {
        SearchDocument {
            key: key.to_owned(),
            id: id.to_owned(),
            kind: kind.to_owned(),
            library_ids: vec![library_id],
            primary: primary.to_owned(),
            secondary: String::new(),
        }
    }

    #[test]
    fn exact_name_outranks_partial_match() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![
            document("artist:1", "1", "artist", 1, "Muse Tribute"),
            document("artist:2", "2", "artist", 1, "Muse"),
        ])?;
        let hits = engine.search("Muse", "artist", &[1], 0, 10)?;
        assert_eq!(hits.first().map(|hit| hit.id.as_str()), Some("2"));
        Ok(())
    }

    #[test]
    fn finds_korean_substrings_and_applies_library_scope() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![
            document("artist:1", "1", "artist", 1, "방탄소년단"),
            document("artist:2", "2", "artist", 2, "소년공화국"),
        ])?;
        let hits = engine.search("소년", "artist", &[1], 0, 10)?;
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].id, "1");
        Ok(())
    }

    #[test]
    fn tolerates_common_latin_transposition_without_beating_exact_match() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![
            document("artist:1", "1", "artist", 1, "Beatles Tribute"),
            document("artist:2", "2", "artist", 1, "Beatles"),
        ])?;
        let typo_hits = engine.search("Beatels", "artist", &[1], 0, 10)?;
        assert!(!typo_hits.is_empty());
        let exact_hits = engine.search("Beatles", "artist", &[1], 0, 10)?;
        assert_eq!(exact_hits.first().map(|hit| hit.id.as_str()), Some("2"));
        Ok(())
    }

    #[test]
    fn upsert_and_delete_are_visible_after_commit() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![document("song:1", "1", "song", 1, "Old Name")])?;
        engine.upsert(vec![document("song:1", "1", "song", 1, "New Name")])?;
        assert_eq!(engine.search("Old", "song", &[1], 0, 10)?.len(), 1);
        assert!(engine.search("New", "song", &[1], 0, 10)?.is_empty());
        engine.commit()?;
        assert!(engine.search("Old", "song", &[1], 0, 10)?.is_empty());
        assert_eq!(engine.search("New", "song", &[1], 0, 10)?.len(), 1);
        engine.delete(vec!["song:1".to_owned()])?;
        assert_eq!(engine.search("New", "song", &[1], 0, 10)?.len(), 1);
        engine.commit()?;
        assert!(engine.search("New", "song", &[1], 0, 10)?.is_empty());
        Ok(())
    }

    #[test]
    fn upsert_rejects_replace_in_progress() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.begin_replace()?;
        let err = engine
            .upsert(vec![document("song:1", "1", "song", 1, "Name")])
            .expect_err("upsert must fail during replacement");
        assert!(err.to_string().contains("replacing"));
        engine.abort_replace()?;
        Ok(())
    }

    #[test]
    fn chunked_replacement_is_atomic_for_searchers() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![document("song:old", "old", "song", 1, "Old Song")])?;

        engine.begin_replace()?;
        engine.append(vec![document("song:new", "new", "song", 1, "New Song")])?;
        assert_eq!(engine.search("Old", "song", &[1], 0, 10)?.len(), 1);
        assert!(engine.search("New", "song", &[1], 0, 10)?.is_empty());

        engine.commit_replace()?;
        assert!(engine.search("Old", "song", &[1], 0, 10)?.is_empty());
        assert_eq!(engine.search("New", "song", &[1], 0, 10)?.len(), 1);
        Ok(())
    }

    #[test]
    fn normalization_preserves_unicode_words() {
        assert_eq!(normalize("  AC/DC — 소년! "), "ac dc 소년");
    }

    #[test]
    fn finds_single_cjk_character_exact_match() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![document("artist:1", "1", "artist", 1, "한")])?;
        let hits = engine.search("한", "artist", &[1], 0, 10)?;
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].id, "1");
        Ok(())
    }

    #[test]
    fn rejects_non_searchable_queries() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![document("artist:1", "1", "artist", 1, "Muse")])?;
        assert!(engine.search("a", "artist", &[1], 0, 10)?.is_empty());
        assert!(engine.search("!!!", "artist", &[1], 0, 10)?.is_empty());
        Ok(())
    }

    #[test]
    fn search_all_returns_scoped_groups_in_one_response() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![
            document("song:1", "song-1", "song", 1, "Blue Monday"),
            document("album:1", "album-1", "album", 1, "Blue Monday"),
            document("artist:1", "artist-1", "artist", 2, "Blue Monday"),
        ])?;

        let response = handle_read_request(&engine, Request::SearchAll {
            query: "Blue".to_owned(),
            library_ids: vec![1],
            searches: vec![
                SearchSpec {
                    kind: "song".to_owned(),
                    offset: 0,
                    limit: 10,
                },
                SearchSpec {
                    kind: "album".to_owned(),
                    offset: 0,
                    limit: 10,
                },
                SearchSpec {
                    kind: "artist".to_owned(),
                    offset: 0,
                    limit: 10,
                },
            ],
        })?;

        assert_eq!(response.groups.len(), 3);
        assert_eq!(response.groups[0].hits[0].id, "song-1");
        assert_eq!(response.groups[1].hits[0].id, "album-1");
        assert!(response.groups[2].hits.is_empty());
        Ok(())
    }

    #[test]
    fn rejects_oversized_search_queries() -> Result<()> {
        let mut engine = Engine::new()?;
        let request = Request::SearchAll {
            query: "a".repeat(MAX_QUERY_CHARS + 1),
            library_ids: Vec::new(),
            searches: Vec::new(),
        };
        assert!(handle_request(&mut engine, request).is_err());
        Ok(())
    }

    #[test]
    fn enrich_secondary_appends_fts_normalize_variants() {
        let enriched = enrich_secondary_with_fts("R.E.M.", "Bjørk album");
        assert!(enriched.contains("REM"), "enriched={enriched}");
        assert!(enriched.contains("Bjork"), "enriched={enriched}");
        assert!(enriched.contains("Bjørk album"), "enriched={enriched}");
    }

    #[test]
    fn apply_indexes_fts_variants_without_pre_normalized_secondary() -> Result<()> {
        let mut engine = Engine::new()?;
        engine.replace(vec![SearchDocument {
            key: "song:1".to_owned(),
            id: "1".to_owned(),
            kind: "song".to_owned(),
            library_ids: vec![1],
            primary: "R.E.M.".to_owned(),
            // Raw secondary only — no Go/metadata search_normalized tokens.
            secondary: "Bjørk Automatic".to_owned(),
        }])?;
        // Tantivy path lowercases after fts-normalize; query normalize strips punct.
        let rem = engine.search("REM", "song", &[1], 0, 10)?;
        assert_eq!(rem.len(), 1, "expected REM via in-Apply fts-normalize");
        let bjork = engine.search("Bjork", "song", &[1], 0, 10)?;
        assert_eq!(bjork.len(), 1, "expected Bjork via in-Apply fts-normalize");
        Ok(())
    }
}
