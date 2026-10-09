//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_generate_rsa4096(t: &Rc<Task>) {
    crypto_call(t, "generate_rsa4096", vec![], V::Nil);
}
