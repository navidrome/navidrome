use std::collections::HashMap;
use std::path::PathBuf;

use anyhow::Result;
use navidrome_grpc_listen::{bind_tcp, local_ipc_server, shutdown, LOCAL_MAX_MSG};
use navidrome_metadata::proto::metadata_server::{Metadata, MetadataServer};
use navidrome_metadata::proto::{
    BuildFts5QueryRequest, BuildFts5QueryResponse, CleanTagsRequest, CleanTagsResponse,
    ExtractFile, ExtractPictureRequest, ExtractPictureResponse, ExtractRequest, ExtractResponse,
    ExtractedMetadata, FileInfo as ProtoFileInfo, HealthRequest, HealthResponse, ImageRequest,
    ImageResponse, MapMediaRequest, MapMediaResponse, NormalizeFtsBatchRequest,
    NormalizeFtsBatchResponse, NormalizeFtsRequest, NormalizeFtsResponse, ParseLyricsRequest,
    ParseLyricsResponse, StringList, TagMapping,
};
use navidrome_metadata::tag_clean::TagMappingConfig;
use tonic::{Request, Response, Status};

use crate::image_worker::{self, ImageOutcome};
use crate::{handle_request, picture_data, read_file, InputFile, Request as ExtractIn};

pub struct MetadataService;

#[tonic::async_trait]
impl Metadata for MetadataService {
    async fn extract(
        &self,
        request: Request<ExtractRequest>,
    ) -> Result<Response<ExtractResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || extract_sync(req))
            .await
            .map_err(|err| Status::internal(format!("extract join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn clean_tags(
        &self,
        request: Request<CleanTagsRequest>,
    ) -> Result<Response<CleanTagsResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || clean_tags_sync(req))
            .await
            .map_err(|err| Status::internal(format!("clean_tags join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn map_media(
        &self,
        request: Request<MapMediaRequest>,
    ) -> Result<Response<MapMediaResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || map_media_sync(req))
            .await
            .map_err(|err| Status::internal(format!("map_media join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn parse_lyrics(
        &self,
        request: Request<ParseLyricsRequest>,
    ) -> Result<Response<ParseLyricsResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || parse_lyrics_sync(req))
            .await
            .map_err(|err| Status::internal(format!("parse_lyrics join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn process_image(
        &self,
        request: Request<ImageRequest>,
    ) -> Result<Response<ImageResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || process_image_sync(req))
            .await
            .map_err(|err| Status::internal(format!("process_image join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn extract_picture(
        &self,
        request: Request<ExtractPictureRequest>,
    ) -> Result<Response<ExtractPictureResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || extract_picture_sync(req))
            .await
            .map_err(|err| Status::internal(format!("extract_picture join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn normalize_fts(
        &self,
        request: Request<NormalizeFtsRequest>,
    ) -> Result<Response<NormalizeFtsResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || normalize_fts_sync(req))
            .await
            .map_err(|err| Status::internal(format!("normalize_fts join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn normalize_fts_batch(
        &self,
        request: Request<NormalizeFtsBatchRequest>,
    ) -> Result<Response<NormalizeFtsBatchResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || normalize_fts_batch_sync(req))
            .await
            .map_err(|err| Status::internal(format!("normalize_fts_batch join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn build_fts5_query(
        &self,
        request: Request<BuildFts5QueryRequest>,
    ) -> Result<Response<BuildFts5QueryResponse>, Status> {
        let req = request.into_inner();
        let result = tokio::task::spawn_blocking(move || build_fts5_query_sync(req))
            .await
            .map_err(|err| Status::internal(format!("build_fts5_query join: {err}")))?;
        Ok(Response::new(result))
    }

    async fn health(
        &self,
        _request: Request<HealthRequest>,
    ) -> Result<Response<HealthResponse>, Status> {
        Ok(Response::new(HealthResponse { ok: true }))
    }
}

pub async fn serve(listen: String) -> Result<()> {
    let service = MetadataServer::new(MetadataService)
        .max_decoding_message_size(LOCAL_MAX_MSG)
        .max_encoding_message_size(LOCAL_MAX_MSG);
    if let Some(path) = listen.strip_prefix("unix:") {
        #[cfg(unix)]
        {
            let incoming = navidrome_grpc_listen::bind_unix(path).await?;
            return local_ipc_server()
                .add_service(service)
                .serve_with_incoming_shutdown(incoming, shutdown())
                .await
                .map_err(|err| anyhow::anyhow!(err));
        }
        #[cfg(not(unix))]
        {
            let _ = path;
            anyhow::bail!("unix sockets are not supported on this platform");
        }
    }
    let incoming = bind_tcp(&listen).await?;
    local_ipc_server()
        .add_service(service)
        .serve_with_incoming_shutdown(incoming, shutdown())
        .await
        .map_err(|err| anyhow::anyhow!(err))
}

fn extract_sync(req: ExtractRequest) -> ExtractResponse {
    let pid_config = if req.pid_config_json.trim().is_empty() {
        None
    } else {
        serde_json::from_str(&req.pid_config_json).ok()
    };
    let request = ExtractIn {
        files: req
            .files
            .into_iter()
            .map(|ExtractFile { key, path }| InputFile {
                key,
                path: PathBuf::from(path),
            })
            .collect(),
        tag_mappings: to_tag_mappings(req.tag_mappings),
        artist_split_exceptions: req.artist_split_exceptions,
        artists_split: req.artists_split,
        roles_split: req.roles_split,
        artist_joiner: req.artist_joiner,
        pid_config,
        library_id: req.library_id,
    };
    let response = handle_request(request);
    ExtractResponse {
        protocol: response.protocol,
        lofty: response.lofty.to_owned(),
        results: response
            .results
            .into_iter()
            .map(|(key, meta)| {
                (
                    key,
                    ExtractedMetadata {
                        tags: to_proto_tags(meta.tags),
                        file_info: Some(ProtoFileInfo {
                            name: meta.file_info.name,
                            size: meta.file_info.size,
                            modified_ns: meta.file_info.modified_ns,
                            created_ns: meta.file_info.created_ns.unwrap_or(0),
                            has_created_ns: meta.file_info.created_ns.is_some(),
                        }),
                        duration_ns: meta.duration_ns,
                        bit_rate: meta.bit_rate,
                        bit_depth: u32::from(meta.bit_depth),
                        sample_rate: meta.sample_rate,
                        channels: u32::from(meta.channels),
                        codec: meta.codec,
                        has_picture: meta.has_picture,
                        lyrics_json: meta.lyrics_json.unwrap_or_default(),
                        media_file_json: meta.media_file_json.unwrap_or_default(),
                        cleaned_tags: to_proto_tags(meta.cleaned_tags.unwrap_or_default()),
                    },
                )
            })
            .collect(),
        errors: response.errors,
    }
}

fn clean_tags_sync(req: CleanTagsRequest) -> CleanTagsResponse {
    let cleaned = navidrome_metadata::tag_clean::clean(
        &req.path,
        &from_proto_tags(req.tags),
        &to_tag_mappings(req.mappings),
        &req.artist_split_exceptions,
    );
    CleanTagsResponse {
        ok: true,
        tags: to_proto_tags(cleaned),
        error: String::new(),
    }
}

fn map_media_sync(req: MapMediaRequest) -> MapMediaResponse {
    let lyrics = if req.lyrics_json.is_empty() {
        "[]"
    } else {
        &req.lyrics_json
    };
    let pid_config = if req.pid_config_json.trim().is_empty() {
        None
    } else {
        serde_json::from_str(&req.pid_config_json).ok()
    };
    let map_config = navidrome_metadata::map_media::MapMediaConfig {
        artists_split: if req.artists_split.is_empty() {
            navidrome_metadata::map_media::MapMediaConfig::with_defaults().artists_split
        } else {
            req.artists_split
        },
        roles_split: if req.roles_split.is_empty() {
            navidrome_metadata::map_media::MapMediaConfig::with_defaults().roles_split
        } else {
            req.roles_split
        },
        artist_split_exceptions: req.artist_split_exceptions,
        artist_joiner: if req.artist_joiner.is_empty() {
            navidrome_metadata::map_media::MapMediaConfig::with_defaults().artist_joiner
        } else {
            req.artist_joiner
        },
    };
    let tags = from_proto_tags(req.tags);
    let path = PathBuf::from(&req.path);
    match navidrome_metadata::map_media::map_to_json_with_pid(
        &tags,
        &path,
        Some(lyrics),
        pid_config.as_ref(),
        req.library_id,
        &req.path,
        Some(&map_config),
    ) {
        Some(json) => MapMediaResponse {
            ok: true,
            media_file_json: json,
            error: String::new(),
        },
        None => MapMediaResponse {
            ok: false,
            media_file_json: String::new(),
            error: "map_media returned no result (missing title and album)".to_owned(),
        },
    }
}

fn parse_lyrics_sync(req: ParseLyricsRequest) -> ParseLyricsResponse {
    match crate::lyrics::parse_lyrics_external(&req.suffix, &req.lang, &req.contents) {
        Ok(json) => ParseLyricsResponse {
            ok: true,
            lyrics_json: json,
            error: String::new(),
        },
        Err(error) => ParseLyricsResponse {
            ok: false,
            lyrics_json: String::new(),
            error,
        },
    }
}

fn process_image_sync(req: ImageRequest) -> ImageResponse {
    let request = image_worker::ImageRequest {
        mosaic: req.mosaic,
        sniff: req.sniff,
        size: req.size,
        square: req.square,
        fill: req.fill,
        animated_gif: req.animated_gif,
        animated_webp: req.animated_webp,
        animated_png: req.animated_png,
        quality: req.quality.min(100) as u8,
        format: image_worker::OutputFormat::parse(&req.format),
        path: if req.path.trim().is_empty() {
            None
        } else {
            Some(req.path)
        },
        input_size: 0,
        input_sizes: Vec::new(),
    };
    match image_worker::process(request, req.payloads) {
        Ok(ImageOutcome::Bytes(body)) => ImageResponse {
            ok: true,
            body,
            error: String::new(),
            ..Default::default()
        },
        Ok(ImageOutcome::Sniff(flags)) => ImageResponse {
            ok: true,
            animated_gif: flags.animated_gif,
            animated_webp: flags.animated_webp,
            animated_png: flags.animated_png,
            has_animation_flags: true,
            ..Default::default()
        },
        Err(error) => ImageResponse {
            ok: false,
            error: format!("{error:#}"),
            ..Default::default()
        },
    }
}

fn extract_picture_sync(req: ExtractPictureRequest) -> ExtractPictureResponse {
    let path = PathBuf::from(&req.path);
    match read_file(&path).and_then(|(tagged, _, _, _)| {
        let picture = picture_data(&tagged, &path)?;
        if req.max_bytes > 0 && picture.len() as i64 > req.max_bytes {
            anyhow::bail!(
                "embedded artwork exceeds maximum size of {} bytes",
                req.max_bytes
            );
        }
        Ok(picture.to_vec())
    }) {
        Ok(body) => ExtractPictureResponse {
            ok: true,
            body,
            error: String::new(),
        },
        Err(error) => ExtractPictureResponse {
            ok: false,
            body: Vec::new(),
            error: format!("{error:#}"),
        },
    }
}

fn normalize_fts_sync(req: NormalizeFtsRequest) -> NormalizeFtsResponse {
    NormalizeFtsResponse {
        ok: true,
        normalized: crate::normalize_fts::normalize_for_fts(&req.values),
        error: String::new(),
    }
}

fn normalize_fts_batch_sync(req: NormalizeFtsBatchRequest) -> NormalizeFtsBatchResponse {
    use rayon::prelude::*;
    let normalized: Vec<String> = req
        .items
        .into_par_iter()
        .map(|item| crate::normalize_fts::normalize_for_fts(&item.values))
        .collect();
    NormalizeFtsBatchResponse {
        ok: true,
        normalized,
        error: String::new(),
    }
}

fn build_fts5_query_sync(req: BuildFts5QueryRequest) -> BuildFts5QueryResponse {
    let built = fts_normalize::build_fts5_query(&req.query);
    BuildFts5QueryResponse {
        ok: true,
        query: built.query,
        degraded: built.degraded,
        error: String::new(),
    }
}

fn to_tag_mappings(mappings: HashMap<String, TagMapping>) -> HashMap<String, TagMappingConfig> {
    mappings
        .into_iter()
        .map(|(name, mapping)| {
            (
                name,
                TagMappingConfig {
                    aliases: mapping.aliases,
                    tag_type: mapping.r#type,
                    max_length: mapping.max_length.max(0) as usize,
                    split: mapping.split,
                    album: mapping.album,
                },
            )
        })
        .collect()
}

fn to_proto_tags(tags: HashMap<String, Vec<String>>) -> HashMap<String, StringList> {
    tags.into_iter()
        .map(|(k, values)| (k, StringList { values }))
        .collect()
}

fn from_proto_tags(tags: HashMap<String, StringList>) -> HashMap<String, Vec<String>> {
    tags.into_iter()
        .map(|(k, list)| (k, list.values))
        .collect()
}
