//! core.string.from_bytes: string(b) copies the bytes.
use super::*;

pub fn from_bytes(b: V) -> V {
    let V::Slice(h, o, l, _) = b else { fault("slice expected") };
    if h == 0 {
        return s(b"");
    }
    let out: Vec<u8> = with(h, |obj| match obj {
        Obj::Vals(v) => v[o as usize..(o + l) as usize].iter().map(|x| x.i() as u8).collect(),
        _ => fault("slice backing expected"),
    });
    V::Str(Rc::from(out))
}
