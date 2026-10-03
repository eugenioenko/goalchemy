//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_hkdf_sha256(t: &Rc<Task>, a0: V, a1: V, a2: V, a3: V) {
    crypto_call(t, "hkdf_sha256", vec![a0, a1, a2, a3], BYTE_NIL);
}
