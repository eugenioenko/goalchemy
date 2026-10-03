//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_aes256_gcm_decrypt(t: &Rc<Task>, a0: V, a1: V, a2: V, a3: V) {
    crypto_call(t, "aes256_gcm_decrypt", vec![a0, a1, a2, a3], BYTE_NIL);
}
