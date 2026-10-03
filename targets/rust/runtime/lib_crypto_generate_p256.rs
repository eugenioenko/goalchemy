//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_generate_p256(t: &Rc<Task>) {
    crypto_call(t, "generate_p256", vec![], V::Nil);
}
