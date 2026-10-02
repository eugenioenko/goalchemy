//! core.slice.copy: copy(dst, src) with overlap handled as Go does.
use super::*;

pub fn copy(dst: V, src: V, clone: Option<fn(&V) -> V>) -> V {
    let (d, dof, dl, _, bytes) = slice_parts(&dst);
    let (sb, sof, sl, _, src_bytes) = slice_parts(&src);
    let n = dl.min(sl) as usize;
    if n == 0 {
        return V::Int(0);
    }
    if bytes {
        if !src_bytes { fault("byte slice expected"); }
        if d == sb { with(d, |obj| match obj {
            Obj::Bytes(v) => v.copy_within(sof as usize..sof as usize + n, dof as usize),
            _ => fault("byte backing expected"),
        }); } else {
            let tmp = byte_snapshot(sb, sof as usize, n);
            with(d, |obj| match obj {
                Obj::Bytes(v) => v[dof as usize..dof as usize + n].copy_from_slice(&tmp),
                _ => fault("byte backing expected"),
            });
        }
        return V::Int(n as i64);
    }
    let tmp: Vec<V> = with(sb, |obj| match obj {
        Obj::Vals(v) => v[sof as usize..sof as usize + n].to_vec(),
        _ => fault("slice backing expected"),
    });
    let tmp: Vec<V> = match clone {
        Some(f) => tmp.iter().map(f).collect(),
        None => tmp,
    };
    with(d, |obj| match obj {
        Obj::Vals(v) => {
            for (k, e) in tmp.into_iter().enumerate() {
                v[dof as usize + k] = e;
            }
        }
        _ => fault("slice backing expected"),
    });
    V::Int(n as i64)
}

pub fn copy_string(dst: V, src: V) -> V {
    let (d, dof, dl, _, bytes) = slice_parts(&dst);
    let b = src.bytes();
    let n = (dl as usize).min(b.len());
    if n == 0 {
        return V::Int(0);
    }
    with(d, |obj| match obj {
        Obj::Bytes(v) if bytes => v[dof as usize..dof as usize + n].copy_from_slice(&b[..n]),
        Obj::Vals(v) => {
            for k in 0..n {
                v[dof as usize + k] = V::Int(b[k] as i64);
            }
        }
        _ => fault("slice backing expected"),
    });
    V::Int(n as i64)
}
