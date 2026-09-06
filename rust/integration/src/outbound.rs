use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use reqwest::dns::{Name, Resolve, Resolving};
use reqwest::redirect::{Action, Attempt};
use tokio::sync::Mutex;
use tonic::{Request, Response, Status};

use crate::breaker::CircuitBreaker;
use crate::proto::outbound_server::Outbound;
use crate::proto::{
    HealthRequest, HealthResponse, HttpRequest, HttpResponse, SignRequest, SignResponse,
};
use crate::sign::sign_audioscrobbler;
use crate::ssrf::is_safe_artwork_ip;

const DEFAULT_TIMEOUT: Duration = Duration::from_secs(10);
const ARTWORK_TIMEOUT: Duration = Duration::from_secs(5);
const MAX_BODY: usize = 8 * 1024 * 1024;
const MAX_ARTWORK_BODY: usize = 20 * 1024 * 1024;
const USER_AGENT: &str = "NavidromeIntegration/1.0";
const DEST_ARTWORK: &str = "artwork";

#[derive(Clone)]
pub struct OutboundService {
    http: reqwest::Client,
    artwork_http: reqwest::Client,
    breakers: Arc<Mutex<HashMap<String, CircuitBreaker>>>,
}

impl OutboundService {
    pub fn new() -> anyhow::Result<Self> {
        let http = reqwest::Client::builder()
            .user_agent(USER_AGENT)
            .http2_adaptive_window(true)
            .pool_idle_timeout(Duration::from_secs(30))
            .pool_max_idle_per_host(8)
            .connect_timeout(Duration::from_secs(5))
            .timeout(DEFAULT_TIMEOUT)
            .build()?;
        let artwork_http = reqwest::Client::builder()
            .user_agent(USER_AGENT)
            .http2_adaptive_window(true)
            .pool_idle_timeout(Duration::from_secs(30))
            .pool_max_idle_per_host(8)
            .connect_timeout(ARTWORK_TIMEOUT)
            .timeout(ARTWORK_TIMEOUT)
            .no_proxy()
            .redirect(reqwest::redirect::Policy::custom(artwork_redirect))
            .dns_resolver(Arc::new(PublicOnlyResolver))
            .build()?;
        Ok(Self {
            http,
            artwork_http,
            breakers: Arc::new(Mutex::new(HashMap::new())),
        })
    }

    fn client_for(&self, dest: &str) -> &reqwest::Client {
        if dest == DEST_ARTWORK {
            &self.artwork_http
        } else {
            &self.http
        }
    }

    fn max_body(dest: &str) -> usize {
        if dest == DEST_ARTWORK {
            MAX_ARTWORK_BODY
        } else {
            MAX_BODY
        }
    }

    async fn breaker_allow(&self, dest: &str) -> bool {
        let mut map = self.breakers.lock().await;
        if let Some(breaker) = map.get_mut(dest) {
            return breaker.allow();
        }
        map.entry(dest.to_owned()).or_default().allow()
    }

    async fn breaker_success(&self, dest: &str) {
        let mut map = self.breakers.lock().await;
        if let Some(breaker) = map.get_mut(dest) {
            breaker.success();
            return;
        }
        map.entry(dest.to_owned()).or_default().success();
    }

    async fn breaker_failure(&self, dest: &str) {
        let mut map = self.breakers.lock().await;
        if let Some(breaker) = map.get_mut(dest) {
            breaker.failure();
            return;
        }
        map.entry(dest.to_owned()).or_default().failure();
    }
}

fn artwork_redirect(attempt: Attempt<'_>) -> Action {
    if attempt.previous().len() >= 10 {
        return attempt.error(std::io::Error::other("stopped after 10 redirects"));
    }
    let scheme = attempt.url().scheme().to_owned();
    if scheme != "http" && scheme != "https" {
        return attempt.error(std::io::Error::other(format!(
            "unsupported artwork redirect scheme {scheme}"
        )));
    }
    // reqwest skips custom DNS for IP literals, so reject unsafe hosts here too.
    if let Some(host) = attempt.url().host_str() {
        if let Some(err) = disallowed_literal_host_error(host) {
            return attempt.error(err);
        }
    }
    attempt.follow()
}

struct PublicOnlyResolver;

impl Resolve for PublicOnlyResolver {
    fn resolve(&self, name: Name) -> Resolving {
        Box::pin(async move {
            let host = name.as_str().trim_end_matches('.');
            lookup_public(host).await
        })
    }
}

async fn lookup_public(
    host: &str,
) -> Result<Box<dyn Iterator<Item = SocketAddr> + Send>, Box<dyn std::error::Error + Send + Sync>>
{
    let addrs = match tokio::net::lookup_host((host, 0)).await {
        Ok(iter) => iter.collect::<Vec<_>>(),
        Err(_) => tokio::net::lookup_host((host, 80)).await?.collect(),
    };
    let filtered: Vec<SocketAddr> = addrs
        .into_iter()
        .filter(|addr| is_safe_artwork_ip(addr.ip()))
        .map(|mut addr| {
            addr.set_port(0);
            addr
        })
        .collect();
    if filtered.is_empty() {
        return Err(std::io::Error::other(format!(
            "artwork destination {host:?} resolved to disallowed address"
        ))
        .into());
    }
    Ok(Box::new(filtered.into_iter()))
}

/// reqwest's custom DNS resolver is not consulted for URL IP literals
/// (`http://127.0.0.1/...`). Reject those hosts up front, matching Go's
/// dial-time SSRF check in core/integration/ssrf.go.
fn disallowed_literal_host_error(host: &str) -> Option<std::io::Error> {
    let host = host.trim_matches(|c| c == '[' || c == ']');
    let Ok(ip) = host.parse::<std::net::IpAddr>() else {
        return None;
    };
    if is_safe_artwork_ip(ip) {
        return None;
    }
    Some(std::io::Error::other(format!(
        "artwork destination {host:?} resolved to disallowed address"
    )))
}

fn validate_artwork_url(url: &str) -> Result<(), Status> {
    let parsed = reqwest::Url::parse(url)
        .map_err(|err| Status::invalid_argument(format!("invalid artwork url: {err}")))?;
    match parsed.scheme() {
        "http" | "https" => {}
        other => {
            return Err(Status::invalid_argument(format!(
                "unsupported artwork scheme {other}"
            )));
        }
    }
    if let Some(host) = parsed.host_str() {
        if let Some(err) = disallowed_literal_host_error(host) {
            return Err(Status::invalid_argument(err.to_string()));
        }
    }
    Ok(())
}

#[tonic::async_trait]
impl Outbound for OutboundService {
    async fn call(&self, request: Request<HttpRequest>) -> Result<Response<HttpResponse>, Status> {
        let req = request.into_inner();
        if req.url.is_empty() {
            return Err(Status::invalid_argument("url is required"));
        }
        let dest = if req.destination.is_empty() {
            "unknown"
        } else {
            req.destination.as_str()
        };
        let max_req_body = Self::max_body(dest);
        if req.body.len() > max_req_body {
            return Err(Status::resource_exhausted("request body too large"));
        }
        if dest == DEST_ARTWORK {
            validate_artwork_url(&req.url)?;
        }
        if !self.breaker_allow(dest).await {
            return Ok(Response::new(HttpResponse {
                error: format!("circuit open for {dest}"),
                ..Default::default()
            }));
        }

        let method =
            reqwest::Method::from_bytes(req.method.as_bytes()).unwrap_or(reqwest::Method::GET);
        let timeout = if req.timeout_ms > 0 {
            Duration::from_millis(req.timeout_ms as u64)
        } else if dest == DEST_ARTWORK {
            ARTWORK_TIMEOUT
        } else {
            DEFAULT_TIMEOUT
        };
        let mut builder = self
            .client_for(dest)
            .request(method, &req.url)
            .timeout(timeout);
        for (key, value) in &req.headers {
            if key.eq_ignore_ascii_case("host") || key.eq_ignore_ascii_case("content-length") {
                continue;
            }
            builder = builder.header(key, value);
        }
        if !req.body.is_empty() {
            builder = builder.body(req.body);
        }

        match builder.send().await {
            Ok(resp) => {
                let status = resp.status().as_u16() as i32;
                let retry_after_ms = parse_retry_after(resp.headers());
                let headers = header_map(resp.headers());
                let max_body = Self::max_body(dest);
                if let Some(len) = resp.content_length() {
                    if len as usize > max_body {
                        self.breaker_failure(dest).await;
                        return Ok(Response::new(HttpResponse {
                            error: format!("response exceeds size limit of {max_body} bytes"),
                            ..Default::default()
                        }));
                    }
                }
                let body = match resp.bytes().await {
                    Ok(bytes) => bytes,
                    Err(err) => {
                        self.breaker_failure(dest).await;
                        return Ok(Response::new(HttpResponse {
                            error: err.to_string(),
                            ..Default::default()
                        }));
                    }
                };
                if body.len() > max_body {
                    self.breaker_failure(dest).await;
                    return Ok(Response::new(HttpResponse {
                        error: format!("response exceeds size limit of {max_body} bytes"),
                        ..Default::default()
                    }));
                }
                if status >= 500 {
                    self.breaker_failure(dest).await;
                } else {
                    self.breaker_success(dest).await;
                }
                Ok(Response::new(HttpResponse {
                    status,
                    headers,
                    body: body.to_vec(),
                    error: String::new(),
                    retry_after_ms,
                }))
            }
            Err(err) => {
                self.breaker_failure(dest).await;
                Ok(Response::new(HttpResponse {
                    error: err.to_string(),
                    ..Default::default()
                }))
            }
        }
    }

    async fn sign(&self, request: Request<SignRequest>) -> Result<Response<SignResponse>, Status> {
        let req = request.into_inner();
        if req.secret.is_empty() {
            return Ok(Response::new(SignResponse {
                error: "secret is required".into(),
                ..Default::default()
            }));
        }
        Ok(Response::new(SignResponse {
            api_sig: sign_audioscrobbler(&req.params, &req.secret),
            error: String::new(),
        }))
    }

    async fn health(
        &self,
        _request: Request<HealthRequest>,
    ) -> Result<Response<HealthResponse>, Status> {
        Ok(Response::new(HealthResponse {
            ok: true,
            version: env!("CARGO_PKG_VERSION").into(),
        }))
    }
}

fn header_map(headers: &reqwest::header::HeaderMap) -> HashMap<String, String> {
    let mut out = HashMap::new();
    for (key, value) in headers {
        if let Ok(v) = value.to_str() {
            out.insert(key.to_string(), v.to_string());
        }
    }
    out
}

fn parse_retry_after(headers: &reqwest::header::HeaderMap) -> i32 {
    let Some(value) = headers.get("retry-after").and_then(|v| v.to_str().ok()) else {
        return 0;
    };
    value.parse::<i32>().ok().unwrap_or(0).saturating_mul(1000)
}

#[cfg(test)]
mod literal_host_tests {
    use super::*;

    #[test]
    fn rejects_literal_loopback_host() {
        let err = validate_artwork_url("http://127.0.0.1:9/secret").unwrap_err();
        assert!(
            err.message().contains("disallowed address"),
            "unexpected: {err}"
        );
        let err = validate_artwork_url("http://[::1]:9/secret").unwrap_err();
        assert!(
            err.message().contains("disallowed address"),
            "unexpected: {err}"
        );
    }

    #[test]
    fn allows_public_literal_and_hostname() {
        validate_artwork_url("http://8.8.8.8/img").expect("public literal");
        validate_artwork_url("https://cdn.example/cover.jpg").expect("hostname");
    }
}
