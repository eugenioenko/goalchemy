//! core.slice.to_array: [N]T(s) copies the first N elements.
use super::*;

pub fn slice_to_array(x: V, n: usize, clone: Option<fn(&V) -> V>) -> V {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    if (l as usize) < n {
        runtime_panic(&format!("cannot convert slice with length {} to array or pointer to array with length {}", l, n));
    }
    let vs: Vec<V> = if n == 0 {
        Vec::new()
    } else {
        with(h, |obj| match obj {
            Obj::Vals(v) => v[o as usize..o as usize + n].to_vec(),
            _ => fault("slice backing expected"),
        })
    };
    let vs = match clone {
        Some(f) => vs.iter().map(f).collect(),
        None => vs,
    };
    vals(vs)
}
