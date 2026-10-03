//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_generate_rsa2048(t: &Rc<Task>) {
    crypto_call(t, "generate_rsa2048", vec![], V::Nil);
}
