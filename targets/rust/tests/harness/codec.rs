//! Canonical value encoding for the Rust harness.
use crate::json::J;
use crate::rt::*;

fn is_nil(raw: &J) -> bool {
    raw.get("nil").map_or(false, |v| v.truthy())
}

fn num(raw: &J) -> i128 {
    raw.str().parse::<i128>().unwrap_or(0)
}

pub fn dec_int(raw: &J, _kind: &str) -> V {
    V::Int(num(raw) as u64 as i64)
}

pub fn dec_bool(raw: &J) -> V {
    V::Bool(matches!(raw, J::Bool(true)) || raw.str() == "true")
}

fn hex(h: &str) -> Vec<u8> {
    (0..h.len() / 2).map(|k| u8::from_str_radix(&h[2 * k..2 * k + 2], 16).unwrap_or(0)).collect()
}

pub fn dec_string(raw: &J) -> V {
    if let Some(x) = raw.get("str") {
        return s(x.str().as_bytes());
    }
    s(&hex(raw.get("hex").map_or("", |x| x.str())))
}

pub fn dec_error(raw: &J) -> V {
    if is_nil(raw) {
        return V::Nil;
    }
    std_errors_new(s(raw.get("error").map_or("", |x| x.str()).as_bytes()))
}

pub fn dec_slice(raw: &J, dec: &dyn Fn(&J) -> V, zero: fn() -> V) -> V {
    if is_nil(raw) {
        return NIL_SLICE;
    }
    let items = raw.get("slice").map_or(&[][..], |x| x.arr());
    let cap = raw.get("cap").map_or(items.len(), |c| num(c) as usize);
    let v: Vec<V> = (0..cap).map(|k| if k < items.len() { dec(&items[k]) } else { zero() }).collect();
    V::Slice(alloc(Obj::Vals(v)), 0, items.len() as u32, cap as u32)
}

pub fn dec_byte_slice(raw: &J) -> V {
    if is_nil(raw) { return BYTE_NIL; }
    let items = raw.get("slice").map_or(&[][..], |x| x.arr());
    let cap = raw.get("cap").map_or(items.len(), |c| num(c) as usize);
    let mut v = vec![0; cap];
    for (k, item) in items.iter().enumerate() { v[k] = num(item) as u8; }
    V::ByteSlice(alloc(Obj::Bytes(v)), 0, items.len() as u32, cap as u32)
}

pub fn view(base: &V, raw: &J) -> V {
    let (a, o, l, c, bytes) = slice_parts(base);
    let lo = raw.get("lo").map_or(0, |x| num(x) as u32);
    let hi = raw.get("hi").map_or(l, |x| num(x) as u32);
    let mx = raw.get("max").map_or(c, |x| num(x) as u32);
    slice_header(a, o + lo, hi - lo, mx - lo, bytes)
}

pub fn dec_map(raw: &J, dk: &dyn Fn(&J) -> V, dv: &dyn Fn(&J) -> V) -> V {
    if is_nil(raw) {
        return V::Nil;
    }
    let m = make_map(vkey);
    for e in raw.get("map").map_or(&[][..], |x| x.arr()) {
        map_set(m.clone(), dk(e.get("key").unwrap()), dv(e.get("value").unwrap()));
    }
    m
}

pub fn dec_chan(raw: &J, dec: &dyn Fn(&J) -> V, zero: fn() -> V) -> V {
    if is_nil(raw) {
        return V::Nil;
    }
    let cap = raw.get("cap").map_or(0, |c| num(c) as i64);
    let ch = make_chan(V::Int(cap), zero);
    let vs: Vec<V> = raw.get("chan").map_or(&[][..], |x| x.arr()).iter().map(|x| dec(x)).collect();
    let closed = raw.get("closed").map_or(false, |x| x.truthy());
    with_chan(&ch, |c| {
        c.buf.extend(vs);
        c.closed = closed;
    });
    ch
}

pub fn enc_int(v: &V, kind: &str) -> J {
    let x = v.i();
    J::Str(if kind == "u64" { (x as u64).to_string() } else { x.to_string() })
}

pub fn enc_bool(v: &V) -> J {
    J::Str(if v.b() { "true" } else { "false" }.into())
}

pub fn enc_string(v: &V) -> J {
    let h: String = v.bytes().iter().map(|b| format!("{:02x}", b)).collect();
    J::obj(vec![("hex", J::Str(h))])
}

pub fn enc_error(v: &V) -> J {
    match unbox(v) {
        None => J::obj(vec![("nil", J::Bool(true))]),
        Some((t, x)) => {
            let (_, code) = t.method("Error").unwrap();
            let msg = code(&[], vec![x]).bytes();
            J::obj(vec![("error", J::Str(String::from_utf8_lossy(&msg).into_owned()))])
        }
    }
}

pub fn enc_slice(v: &V, enc: &dyn Fn(&V) -> J) -> J {
    let (a, o, l, c, _) = slice_parts(v);
    if a == 0 {
        return J::obj(vec![("nil", J::Bool(true))]);
    }
    let items = (0..l).map(|k| enc(&slot(a, (o + k) as usize))).collect();
    J::obj(vec![("slice", J::Arr(items)), ("cap", J::Str(c.to_string()))])
}

pub fn enc_array(v: &V, enc: &dyn Fn(&V) -> J) -> J {
    let items = (0..vals_len(v.h())).map(|i| enc(&slot(v.h(), i))).collect();
    J::obj(vec![("array", J::Arr(items))])
}

pub fn enc_map(v: &V, ek: &dyn Fn(&V) -> J, ev: &dyn Fn(&V) -> J) -> J {
    if v.is_nil() {
        return J::obj(vec![("nil", J::Bool(true))]);
    }
    let it = map_iter(v.clone());
    let mut items = Vec::new();
    while map_next(it.clone()).b() {
        items.push(J::obj(vec![("key", ek(&iter_key(it.clone()))), ("value", ev(&iter_val(it.clone())))]));
    }
    J::obj(vec![("map", J::Arr(items))])
}

pub fn enc_chan(v: &V, enc: &dyn Fn(&V) -> J) -> J {
    if v.is_nil() {
        return J::obj(vec![("nil", J::Bool(true))]);
    }
    let (buf, size, closed) = with_chan(v, |c| (c.buf.iter().cloned().collect::<Vec<V>>(), c.size, c.closed));
    let items = buf.iter().map(|x| enc(x)).collect();
    J::obj(vec![("chan", J::Arr(items)), ("cap", J::Str(size.to_string())), ("closed", J::Bool(closed))])
}

pub fn enc_zero(_: &V) -> J {
    J::obj(vec![("zero", J::Bool(true))])
}

fn mark(e: &[V], _: Vec<V>) -> V {
    set_slot(e[0].h(), 0, V::Bool(true));
    V::Nil
}

/// Reports whether a spawned task ran before the parent yielded.
fn spawn_check_step(t: &Rc<Task>, f: &Rc<Frame>) {
    if f.pc.get() == 0 {
        let flag = vals(vec![V::Bool(false)]);
        f.l.s(0, flag.clone());
        spawn(sync_frame(func(-1, mark, vec![flag.clone()]), Vec::new(), 0));
        f.l.s(1, slot(flag.h(), 0));
        f.pc.set(1);
        std_runtime_gosched(t);
        return;
    }
    if !slot(f.l.g(0).h(), 0).b() {
        panic!("spawned task did not run after the parent yielded");
    }
    ret(t, f)
}

fn spawn_check_results(f: &Frame) -> Vec<V> {
    vec![f.l.g(1)]
}

pub fn new_spawn_check() -> V {
    V::Frame(Frame::new(2, spawn_check_step, Some(spawn_check_results)))
}

pub fn harness_select2(t: &Rc<Task>, a: V, b: V, dflt: V) {
    select(t, dflt.b(), vec![scase(a, false, V::Nil), scase(b, false, V::Nil)])
}
