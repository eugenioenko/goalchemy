//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_generate_p521(t: &Rc<Task>) {
    crypto_call(t, "generate_p521", vec![], V::Nil);
}
