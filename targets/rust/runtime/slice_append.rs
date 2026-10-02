//! core.slice.append: append with Goalchemy's growth rule; aggregate
//! elements are cloned when they move.
use super::*;

pub fn grow_cap(old: i64, required: i64) -> i64 {
    let doubled = if old <= i64::MAX / 2 { 2 * old } else { i64::MAX };
    required.max(1.max(doubled))
}

fn append_values(x: V, vs: Vec<V>, clone: Option<fn(&V) -> V>) -> V {
    let V::Slice(a, o, l, c) = x else { fault("slice expected") };
    if vs.is_empty() {
        return x;
    }
    let n = l as usize + vs.len();
    if n <= c as usize {
        with(a, |obj| match obj {
            Obj::Vals(v) => {
                for (k, e) in vs.into_iter().enumerate() {
                    v[o as usize + l as usize + k] = e;
                }
            }
            _ => fault("slice backing expected"),
        });
        return V::Slice(a, o, n as u32, c);
    }
    let nc = grow_cap(c as i64, n as i64);
    if nc > u32::MAX as i64 / 2 {
        fault("slice growth exceeds host limits");
    }
    let old: Vec<V> = if a == 0 {
        Vec::new()
    } else {
        with(a, |obj| match obj {
            Obj::Vals(v) => v[o as usize..(o + l) as usize].to_vec(),
            _ => fault("slice backing expected"),
        })
    };
    let mut nv: Vec<V> = Vec::with_capacity(nc as usize);
    for e in old.iter() {
        nv.push(match clone {
            Some(f) => f(e),
            None => e.clone(),
        });
    }
    nv.extend(vs);
    nv.resize(nc as usize, V::Nil);
    V::Slice(alloc(Obj::Vals(nv)), 0, n as u32, nc as u32)
}

pub fn append(x: V, vs: Vec<V>, clone: Option<fn(&V) -> V>) -> V {
    append_values(x, vs, clone)
}

pub fn append_slice(x: V, y: V, clone: Option<fn(&V) -> V>) -> V {
    if matches!(x, V::ByteSlice(..)) {
        let (h, o, l, _, bytes) = slice_parts(&y);
        if !bytes { fault("byte slice expected"); }
        if l == 0 { return x; }
        return append_bytes(x, &byte_snapshot(h, o as usize, l as usize));
    }
    let V::Slice(b, o, l, _) = y else { fault("slice expected") };
    if l == 0 {
        let V::Slice(..) = x else { fault("slice expected") };
        return x;
    }
    let vs: Vec<V> = with(b, |obj| match obj {
        Obj::Vals(v) => v[o as usize..(o + l) as usize].to_vec(),
        _ => fault("slice backing expected"),
    });
    let vs = match clone {
        Some(f) => vs.iter().map(f).collect(),
        None => vs,
    };
    append_values(x, vs, clone)
}

pub fn append_string(x: V, y: V) -> V {
    if matches!(x, V::ByteSlice(..)) { return append_bytes(x, &y.bytes()); }
    let vs: Vec<V> = y.bytes().iter().map(|&c| V::Int(c as i64)).collect();
    append_values(x, vs, None)
}


/// Explicit values, spreads and strings all enter through native bytes.
/// Snapshots for spreads permit overlap in either direction without heap reentry.
pub fn append_bytes(x: V, vs: &[u8]) -> V {
    let (a, o, l, c, bytes) = slice_parts(&x);
    if !bytes { fault("byte slice expected"); }
    if vs.is_empty() { return x; }
    let n = (l as usize).checked_add(vs.len()).unwrap_or_else(|| fault("slice growth exceeds host limits"));
    if n > u32::MAX as usize / 2 { fault("slice growth exceeds host limits"); }
    if n <= c as usize {
        with(a, |obj| match obj {
            Obj::Bytes(v) => v[o as usize + l as usize..o as usize + n].copy_from_slice(vs),
            _ => fault("byte backing expected"),
        });
        return V::ByteSlice(a, o, n as u32, c);
    }
    let nc = grow_cap(c as i64, n as i64);
    if nc > u32::MAX as i64 / 2 { fault("slice growth exceeds host limits"); }
    let mut nv = vec![0; nc as usize];
    if l != 0 { with(a, |obj| match obj {
        Obj::Bytes(v) => nv[..l as usize].copy_from_slice(&v[o as usize..(o + l) as usize]),
        _ => fault("byte backing expected"),
    }); }
    nv[l as usize..n].copy_from_slice(vs);
    V::ByteSlice(alloc(Obj::Bytes(nv)), 0, n as u32, nc as u32)
}
