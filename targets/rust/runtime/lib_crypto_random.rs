//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_random(t: &Rc<Task>, a0: V) {
    crypto_call(t, "random", vec![a0], BYTE_NIL);
}
