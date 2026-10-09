//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_rs512_sign(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "rs512_sign", vec![a0, a1], BYTE_NIL);
}
