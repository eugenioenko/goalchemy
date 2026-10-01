//! std.errors.unwrap.
use super::*;

pub fn std_errors_unwrap(err: V) -> V {
    let Some((t, v)) = unbox(&err) else { return V::Nil };
    match t.method("Unwrap") {
        Some((_, f)) => f(&[], vec![v]),
        None => V::Nil,
    }
}
