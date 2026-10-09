//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_generate_p384(t: &Rc<Task>) {
    crypto_call(t, "generate_p384", vec![], V::Nil);
}
