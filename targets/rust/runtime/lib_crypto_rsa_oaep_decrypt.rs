//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_rsa_oaep_decrypt(t: &Rc<Task>, a0: V, a1: V) {
    crypto_call(t, "rsa_oaep_decrypt", vec![a0, a1], BYTE_NIL);
}
