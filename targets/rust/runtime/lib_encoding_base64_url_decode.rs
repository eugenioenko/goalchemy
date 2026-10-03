//! Bounded canonical RFC4648 encoding.
#![cfg(feature = "native")]
use super::*;
pub fn lib_encoding_base64_url_decode(v: V) -> V {
    native_encoding(v, true, true)
}
