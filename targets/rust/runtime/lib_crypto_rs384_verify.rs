//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_rs384_verify(t: &Rc<Task>, a0: V, a1: V, a2: V) {
    crypto_call(t, "rs384_verify", vec![a0, a1, a2], V::Bool(false));
}
