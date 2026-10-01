//! core.string.from_runes: string(runes) encodes each code point.
use super::*;

pub fn from_runes(r: V) -> V {
    let V::Slice(h, o, l, _) = r else { fault("slice expected") };
    let mut out = Vec::new();
    if h != 0 {
        let rs: Vec<i64> = with(h, |obj| match obj {
            Obj::Vals(v) => v[o as usize..(o + l) as usize].iter().map(|x| x.i()).collect(),
            _ => fault("slice backing expected"),
        });
        for r in rs {
            utf8::encode(r, &mut out);
        }
    }
    V::Str(Rc::from(out))
}
