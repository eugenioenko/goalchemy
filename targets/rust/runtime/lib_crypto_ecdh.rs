//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_ecdh(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "ecdh", vec![a0, a1], BYTE_NIL);
}
