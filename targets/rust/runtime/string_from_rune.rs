//! core.string.from_rune: string(r) encodes one code point.
use super::*;

pub fn from_rune(x: V) -> V {
    let mut out = Vec::new();
    utf8::encode(x.i(), &mut out);
    V::Str(Rc::from(out))
}

/// For unsigned 64-bit operands: negative values are huge.
pub fn from_rune_u(x: V) -> V {
    let r = x.i();
    from_rune(V::Int(if r < 0 { -1 } else { r }))
}
