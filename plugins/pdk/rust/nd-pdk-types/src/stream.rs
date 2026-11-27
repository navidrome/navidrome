//! Byte streams owned by Navidrome, readable and writable from plugins.

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::io;

#[link(wasm_import_module = "extism:host/user")]
extern "C" {
    fn stream_read(handle: i64, buf: u64, len: i64) -> i64;
    fn stream_write(handle: i64, buf: u64, len: i64) -> i64;
    fn stream_copy(dst: i64, src: i64) -> i64;
    fn stream_close(handle: i64) -> i64;
    fn stream_error() -> i64;
}

const BUFFER_SIZE: usize = 64 * 1024;

/// A handle to a byte stream owned by Navidrome. Dropping it does not close the stream: a stream returned to
/// Navidrome must stay open, and Navidrome closes what is left when the call ends.
#[derive(Debug)]
pub struct Stream {
    handle: u32,
    buf: Option<u64>,
}

impl Stream {
    pub fn from_handle(handle: u32) -> Self {
        Stream { handle, buf: None }
    }

    pub fn handle(&self) -> u32 {
        self.handle
    }

    pub fn close(mut self) -> io::Result<()> {
        self.free_buffer();
        if unsafe { stream_close(self.handle as i64) } < 0 {
            return Err(last_error());
        }
        Ok(())
    }

    /// Extism returns offset 0 when memory runs out, and storing there would corrupt its block memory.
    fn buffer(&mut self) -> io::Result<u64> {
        if let Some(offset) = self.buf {
            return Ok(offset);
        }
        let offset = unsafe { extism_pdk::extism::alloc(BUFFER_SIZE as u64) };
        if offset == 0 {
            return Err(io::Error::new(
                io::ErrorKind::OutOfMemory,
                "stream buffer allocation failed",
            ));
        }
        self.buf = Some(offset);
        Ok(offset)
    }

    fn free_buffer(&mut self) {
        if let Some(offset) = self.buf.take() {
            unsafe { extism_pdk::extism::free(offset) }
        }
    }
}

/// Moves all bytes from src to dst inside Navidrome; they never enter plugin memory.
pub fn copy(dst: &Stream, src: &Stream) -> io::Result<u64> {
    let n = unsafe { stream_copy(dst.handle as i64, src.handle as i64) };
    if let Some(err) = pending_error() {
        return Err(err);
    }
    Ok(n as u64)
}

impl io::Read for Stream {
    fn read(&mut self, out: &mut [u8]) -> io::Result<usize> {
        if out.is_empty() {
            return Ok(0);
        }
        let n = out.len().min(BUFFER_SIZE);
        let buf = self.buffer()?;
        let got = unsafe { stream_read(self.handle as i64, buf, n as i64) };
        if got < 0 {
            return Err(last_error());
        }
        let got = got as usize;
        load_bytes(buf, &mut out[..got]);
        Ok(got)
    }
}

impl io::Write for Stream {
    fn write(&mut self, data: &[u8]) -> io::Result<usize> {
        if data.is_empty() {
            return Ok(0);
        }
        let n = data.len().min(BUFFER_SIZE);
        let buf = self.buffer()?;
        store_bytes(buf, &data[..n]);
        let written = unsafe { stream_write(self.handle as i64, buf, n as i64) };
        if written < 0 {
            return Err(last_error());
        }
        Ok(written as usize)
    }

    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl Clone for Stream {
    // Copies share the handle and get their own buffer, so dropping one never frees the other's.
    fn clone(&self) -> Self {
        Stream::from_handle(self.handle)
    }
}

impl Drop for Stream {
    fn drop(&mut self) {
        self.free_buffer();
    }
}

impl Serialize for Stream {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_u32(self.handle)
    }
}

impl<'de> Deserialize<'de> for Stream {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        u32::deserialize(deserializer).map(Stream::from_handle)
    }
}

/// Serde for `Option<Stream>` fields: handle 0 or null means no stream.
pub mod optional {
    use super::Stream;
    use serde::{Deserialize, Deserializer, Serializer};

    pub fn serialize<S: Serializer>(v: &Option<Stream>, s: S) -> Result<S::Ok, S::Error> {
        s.serialize_u32(v.as_ref().map_or(0, Stream::handle))
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Option<Stream>, D::Error> {
        Ok(Option::<u32>::deserialize(d)?
            .filter(|&h| h != 0)
            .map(Stream::from_handle))
    }
}

// extism_pdk::extism::load/store cast the slice to *u64, which needs 8-byte alignment; these copy byte-safely.
fn load_bytes(offset: u64, out: &mut [u8]) {
    let mut chunks = out.chunks_exact_mut(8);
    let mut at = offset;
    for chunk in &mut chunks {
        chunk.copy_from_slice(&unsafe { extism_pdk::extism::load_u64(at) }.to_le_bytes());
        at += 8;
    }
    for byte in chunks.into_remainder() {
        *byte = unsafe { extism_pdk::extism::load_u8(at) };
        at += 1;
    }
}

fn store_bytes(offset: u64, data: &[u8]) {
    let mut chunks = data.chunks_exact(8);
    let mut at = offset;
    for chunk in &mut chunks {
        let word = u64::from_le_bytes(chunk.try_into().expect("chunk of 8 bytes"));
        unsafe { extism_pdk::extism::store_u64(at, word) };
        at += 8;
    }
    for &byte in chunks.remainder() {
        unsafe { extism_pdk::extism::store_u8(at, byte) };
        at += 1;
    }
}

fn pending_error() -> Option<io::Error> {
    let ptr = unsafe { stream_error() };
    if ptr == 0 {
        return None;
    }
    if ptr < 0 {
        return Some(failed());
    }
    let Some(mem) = extism_pdk::Memory::find(ptr as u64) else {
        return Some(failed());
    };
    let err = mem.to_string().map_or_else(|_| failed(), io::Error::other);
    mem.free();
    Some(err)
}

fn last_error() -> io::Error {
    pending_error().unwrap_or_else(failed)
}

fn failed() -> io::Error {
    io::Error::other("stream operation failed")
}
