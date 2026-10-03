//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_sha256(t: &Rc<Task>, a0: V) {
    crypto_call(t, "sha256", vec![a0], BYTE_NIL);
}
