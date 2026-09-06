pub mod proto {
    tonic::include_proto!("navidrome.apikeys.v1");
}

pub mod grpc;

use anyhow::{Context, Result, bail};
use base64::{Engine, engine::general_purpose::URL_SAFE_NO_PAD};
use rand::Rng;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::io::{self, BufRead, BufReader, BufWriter, Write};
use subtle::ConstantTimeEq;

pub const TOKEN_PREFIX: &str = "nd_";
const TOKEN_BYTES: usize = 32;
pub const LOOKUP_PREFIX_LEN: usize = 12;

#[derive(Debug, Deserialize)]
#[serde(tag = "op", rename_all = "lowercase")]
pub enum Request {
    Generate { pepper: String },
    Hash { token: String, pepper: String },
    Verify {
        token: String,
        hash: String,
        pepper: String,
    },
}

#[derive(Debug, Serialize)]
pub struct Response {
    pub ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub token: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lookup_prefix: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub valid: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

pub fn handle_request(request: Request) -> Response {
    match request {
        Request::Generate { pepper } => match generate(&pepper) {
            Ok((token, lookup_prefix, hash)) => Response {
                ok: true,
                token: Some(token),
                lookup_prefix: Some(lookup_prefix),
                hash: Some(hash),
                valid: None,
                error: None,
            },
            Err(error) => failure(error),
        },
        Request::Hash { token, pepper } => match hash_token(&token, &pepper) {
            Ok(hash) => Response {
                ok: true,
                token: None,
                lookup_prefix: Some(lookup_prefix(&token)),
                hash: Some(hash),
                valid: None,
                error: None,
            },
            Err(error) => failure(error),
        },
        Request::Verify { token, hash, pepper } => match verify_token(&token, &hash, &pepper) {
            Ok(valid) => Response {
                ok: true,
                token: None,
                lookup_prefix: None,
                hash: None,
                valid: Some(valid),
                error: None,
            },
            Err(error) => failure(error),
        },
    }
}

fn failure(error: anyhow::Error) -> Response {
    Response {
        ok: false,
        token: None,
        lookup_prefix: None,
        hash: None,
        valid: None,
        error: Some(error.to_string()),
    }
}

pub fn generate(pepper: &str) -> Result<(String, String, String)> {
    validate_pepper(pepper)?;
    let mut raw = [0u8; TOKEN_BYTES];
    rand::rng().fill_bytes(&mut raw);
    let secret = URL_SAFE_NO_PAD.encode(raw);
    let token = format!("{TOKEN_PREFIX}{secret}");
    let lookup = lookup_prefix(&token);
    let hash = hash_token(&token, pepper)?;
    Ok((token, lookup, hash))
}

pub fn lookup_prefix(token: &str) -> String {
    let body = token.strip_prefix(TOKEN_PREFIX).unwrap_or(token);
    body.chars().take(LOOKUP_PREFIX_LEN).collect()
}

pub fn hash_token(token: &str, pepper: &str) -> Result<String> {
    validate_pepper(pepper)?;
    if token.len() < TOKEN_PREFIX.len() + 8 {
        bail!("token is too short");
    }
    let mut hasher = Sha256::new();
    hasher.update(pepper.as_bytes());
    hasher.update(token.as_bytes());
    Ok(hex::encode(hasher.finalize()))
}

pub fn verify_token(token: &str, expected_hash: &str, pepper: &str) -> Result<bool> {
    let actual = hash_token(token, pepper)?;
    Ok(actual.as_bytes().ct_eq(expected_hash.as_bytes()).into())
}

fn validate_pepper(pepper: &str) -> Result<()> {
    if pepper.len() < 16 {
        bail!("pepper must be at least 16 bytes");
    }
    Ok(())
}

pub fn run_worker() -> Result<()> {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut input = BufReader::with_capacity(16 * 1024, stdin.lock());
    let mut output = BufWriter::with_capacity(16 * 1024, stdout.lock());

    loop {
        let mut line = String::new();
        if input.read_line(&mut line)? == 0 {
            break;
        }
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let response = match serde_json::from_str::<Value>(line)
            .context("decoding apikeys request")
            .and_then(|value| {
                serde_json::from_value::<Request>(value).context("parsing apikeys request")
            }) {
            Ok(request) => handle_request(request),
            Err(error) => Response {
                ok: false,
                token: None,
                lookup_prefix: None,
                hash: None,
                valid: None,
                error: Some(error.to_string()),
            },
        };
        serde_json::to_writer(&mut output, &response).context("encoding apikeys response")?;
        output.write_all(b"\n").context("framing apikeys response")?;
        output.flush()?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn round_trip_generate_and_verify() {
        let pepper = "pepper-with-enough-length";
        let (token, lookup, hash) = generate(pepper).expect("generate");
        assert!(token.starts_with(TOKEN_PREFIX));
        assert_eq!(lookup.len(), LOOKUP_PREFIX_LEN);
        assert!(verify_token(&token, &hash, pepper).expect("verify"));
        assert!(!verify_token(&token, "deadbeef", pepper).expect("verify"));
    }
}
