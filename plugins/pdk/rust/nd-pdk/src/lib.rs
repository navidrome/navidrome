//! Navidrome Plugin Development Kit for Rust
//!
//! This crate provides a unified API for building Navidrome plugins in Rust.
//! It re-exports all functionality from the host and capabilities sub-crates.
//!
//! # Example
//!
//! ```rust,no_run
//! use nd_pdk::scrobbler::{Scrobbler, IsAuthorizedRequest, Error};
//! use nd_pdk::register_scrobbler;
//!
//! struct MyPlugin;
//!
//! impl Default for MyPlugin {
//!     fn default() -> Self { MyPlugin }
//! }
//!
//! impl Scrobbler for MyPlugin {
//!     fn is_authorized(&self, req: IsAuthorizedRequest) -> Result<bool, Error> {
//!         Ok(true)
//!     }
//!     // ... implement other required methods
//! }
//!
//! register_scrobbler!(MyPlugin);
//! ```

/// Host function wrappers for calling Navidrome services from plugins.
pub use nd_pdk_host as host;

/// Capability wrappers for implementing plugin exports.
pub use nd_pdk_capabilities::*;

/// Re-export extism-pdk for convenience.
pub use extism_pdk;
