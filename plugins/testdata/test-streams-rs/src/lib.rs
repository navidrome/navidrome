use extism_pdk::*;
use nd_pdk::types::stream::Stream;
use serde::{Deserialize, Serialize};
use std::io::{Read, Write};

#[derive(Deserialize)]
struct Input {
    handle: u32,
}

#[derive(Serialize, Default)]
struct Output {
    count: u64,
    data: String,
    error: String,
}

#[plugin_fn]
pub fn read_all(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut stream = Stream::from_handle(input.handle);
    let mut data = Vec::new();
    let mut out = Output::default();
    if let Err(e) = stream.read_to_end(&mut data) {
        out.error = e.to_string();
    }
    out.count = data.len() as u64;
    out.data = String::from_utf8_lossy(&data).into_owned();
    Ok(Json(out))
}

// Reads into and writes from slices that start at an odd address, to catch alignment bugs.
#[plugin_fn]
pub fn read_unaligned(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut stream = Stream::from_handle(input.handle);
    let mut buf = [0u8; 64];
    let mut out = Output::default();
    match stream.read(&mut buf[1..]) {
        Ok(n) => {
            out.count = n as u64;
            out.data = String::from_utf8_lossy(&buf[1..1 + n]).into_owned();
        }
        Err(e) => out.error = e.to_string(),
    }
    Ok(Json(out))
}

#[plugin_fn]
pub fn write_unaligned(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut stream = Stream::from_handle(input.handle);
    let data = b"_unaligned write, 8+ bytes";
    let mut out = Output::default();
    match stream.write_all(&data[1..]).and_then(|_| stream.close()) {
        Ok(()) => out.count = (data.len() - 1) as u64,
        Err(e) => out.error = e.to_string(),
    }
    Ok(Json(out))
}

#[plugin_fn]
pub fn write_greeting(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut stream = Stream::from_handle(input.handle);
    let greeting = b"hello from rust";
    let mut out = Output::default();
    match stream.write_all(greeting).and_then(|_| stream.close()) {
        Ok(()) => out.count = greeting.len() as u64,
        Err(e) => out.error = e.to_string(),
    }
    Ok(Json(out))
}

/// Fills Extism's block memory before it writes, so the stream buffer cannot be allocated.
#[plugin_fn]
pub fn write_when_out_of_memory(Json(input): Json<Input>) -> FnResult<Json<Output>> {
    let mut blocks = Vec::new();
    for _ in 0..1 << 16 {
        let offset = unsafe { extism_pdk::extism::alloc(64 << 10) };
        if offset == 0 {
            break;
        }
        blocks.push(offset);
    }
    let err = Stream::from_handle(input.handle).write(b"x").err();
    for offset in blocks {
        unsafe { extism_pdk::extism::free(offset) }
    }
    Ok(Json(Output {
        error: err.map(|e| e.to_string()).unwrap_or_default(),
        ..Default::default()
    }))
}
