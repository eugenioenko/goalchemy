//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_hmac_sha256(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "hmac_sha256", vec![a0, a1], BYTE_NIL);
}
