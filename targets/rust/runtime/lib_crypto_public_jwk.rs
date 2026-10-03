//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_public_jwk(t: &Rc<Task>, a0: V) {
    crypto_call(t, "public_jwk", vec![a0], NIL_SLICE);
}
