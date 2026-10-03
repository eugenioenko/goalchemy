//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_private_pem(t: &Rc<Task>, a0: V) {
    crypto_call(t, "private_pem", vec![a0], s(b""));
}
