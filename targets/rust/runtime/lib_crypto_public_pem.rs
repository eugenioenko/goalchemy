//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_public_pem(t: &Rc<Task>, a0: V) {
    crypto_call(t, "public_pem", vec![a0], s(b""));
}
