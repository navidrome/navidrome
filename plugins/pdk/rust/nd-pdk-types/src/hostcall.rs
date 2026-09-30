//! Calls Navidrome host functions and frees both memory blocks, which `#[host_fn]` does not do for the
//! response.

use extism_pdk::{Error, Memory};
use serde::{de::DeserializeOwned, Serialize};

pub fn call<Req: Serialize, Resp: DeserializeOwned>(
    f: unsafe extern "C" fn(u64) -> u64,
    req: &Req,
) -> Result<Resp, Error> {
    let input = Memory::from_bytes(serde_json::to_vec(req)?)?;
    let output = unsafe { f(input.offset()) };
    input.free();
    let mem = Memory::find(output).ok_or_else(|| Error::msg("host function returned no response"))?;
    let bytes = mem.to_vec();
    mem.free();
    Ok(serde_json::from_slice(&bytes)?)
}
