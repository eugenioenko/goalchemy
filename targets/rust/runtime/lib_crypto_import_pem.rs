//! Maintained native capability.
#![cfg(feature = "native")]
use super::*;
pub fn lib_crypto_import_pem(t: &Rc<Task>, a0: V) {
    crypto_call(t, "import_pem", vec![a0], V::Nil);
}
