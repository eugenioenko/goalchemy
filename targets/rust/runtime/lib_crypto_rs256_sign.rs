//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_rs256_sign(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "rs256_sign", vec![a0, a1], BYTE_NIL);
}
