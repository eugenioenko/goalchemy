//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_es384_sign(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "es384_sign", vec![a0, a1], BYTE_NIL);
}
