//! core.string.from_bytes: string(b) copies the bytes.
use super::*;

pub fn from_bytes(b: V) -> V {
    let (h, o, l, _, _) = slice_parts(&b);
    if h == 0 {
        return s(b"");
    }
    let out: Vec<u8> = with(h, |obj| match obj {
        Obj::Bytes(v) => v[o as usize..(o + l) as usize].to_vec(),
        Obj::Vals(v) => v[o as usize..(o + l) as usize].iter().map(|x| x.i() as u8).collect(),
        _ => fault("slice backing expected"),
    });
    V::Str(Rc::from(out))
}
