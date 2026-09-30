use extism_pdk::*;
use serde::{Deserialize, Serialize};
use std::io::Read;

mod streamtest;

#[derive(Deserialize)]
struct Input {
    #[serde(default)]
    text: String,
}

#[derive(Serialize, Default)]
struct Output {
    data: String,
    size: i64,
    error: String,
}

fn read_all(stream: Option<nd_pdk_types::stream::Stream>) -> Result<String, Error> {
    let mut stream = stream.ok_or_else(|| Error::msg("no stream"))?;
    let mut data = String::new();
    stream.read_to_string(&mut data)?;
    Ok(data)
}

#[plugin_fn]
pub fn open_read(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut out = Output::default();
    match streamtest::open(&input.text).and_then(read_all) {
        Ok(data) => out.data = data,
        Err(e) => out.error = e.to_string(),
    }
    Ok(Json(out))
}

// Each call returns a 64 KiB string, so a leaked response block per call runs past the memory cap.
#[plugin_fn]
pub fn echo_loop(Json(_input): Json<Input>) -> FnResult<Json<Output>> {
    let text = "x".repeat(64 * 1024);
    let mut out = Output::default();
    for i in 0..500 {
        match streamtest::echo(&text) {
            Ok(echoed) if echoed.len() == text.len() => out.size = i + 1,
            Ok(_) => {
                out.error = "short echo".to_string();
                break;
            }
            Err(e) => {
                out.error = e.to_string();
                break;
            }
        }
    }
    Ok(Json(out))
}

#[plugin_fn]
pub fn wrap(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut out = Output::default();
    let result = streamtest::open(&input.text).and_then(|body| {
        streamtest::wrap(streamtest::WrapRequest { prefix: "p:".to_string(), body })
    });
    match result {
        Ok(Some(resp)) => {
            out.size = resp.size;
            match read_all(resp.body) {
                Ok(data) => out.data = data,
                Err(e) => out.error = e.to_string(),
            }
        }
        Ok(None) => out.error = "no response".to_string(),
        Err(e) => out.error = e.to_string(),
    }
    Ok(Json(out))
}
