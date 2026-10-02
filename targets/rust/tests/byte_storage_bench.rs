//! Fresh-process diagnostic, not a timing or RSS acceptance threshold.
mod rt;
use rt::*;
use std::time::Instant;
use std::hint::black_box;

fn rss() -> usize {
    std::fs::read_to_string("/proc/self/status").unwrap().lines()
        .find(|s| s.starts_with("VmRSS:")).unwrap().split_whitespace().nth(1).unwrap().parse::<usize>().unwrap() * 1024
}
fn storage(x: &V) -> (usize, usize, usize) {
    let (h, _, _, _, _) = slice_parts(x);
    with(h, |o| match o {
        Obj::Bytes(v) => (1, v.len(), v.capacity()),
        Obj::Vals(v) => (std::mem::size_of::<V>(), v.len(), v.capacity()),
        _ => panic!("storage expected"),
    })
}
fn verify(x: &V, n: usize) {
    let (h, _, l, _, _) = slice_parts(x);
    assert_eq!(l as usize, n);
    with(h, |o| match o {
        Obj::Bytes(v) => { for (i, x) in v[..n].iter().enumerate() { assert_eq!(*x, (i % 256) as u8); } },
        Obj::Vals(v) => { for (i, x) in v[..n].iter().enumerate() { assert_eq!(x.i(), (i % 256) as i64); } },
        _ => panic!("storage expected"),
    });
}
fn main() {
    let args: Vec<String> = std::env::args().collect();
    let mode = &args[1];
    let n = args[2].parse::<usize>().unwrap() * 1024 * 1024;
    let native = mode == "native";
    let f = Fr::new(3);
    let before = rss();
    let start = Instant::now();
    let src = if native {
        V::ByteSlice(alloc(Obj::Bytes((0..n).map(|i| (i % 256) as u8).collect())), 0, n as u32, n as u32)
    } else {
        V::Slice(alloc(Obj::Vals((0..n).map(|i| int((i % 256) as i64)).collect())), 0, n as u32, n as u32)
    };
    f.s(0, src.clone());
    let dst = if native { make_byte_slice(int(n as i64), int(n as i64)) }
        else { make_slice(int(n as i64), int(n as i64), zero_int, false) };
    f.s(1, dst.clone());
    let allocate_ms = start.elapsed().as_secs_f64() * 1000.0;
    let start = Instant::now();
    for _ in 0..16 { assert_eq!(black_box(copy(dst.clone(), src.clone(), None)).i(), n as i64); }
    let copy_ms = start.elapsed().as_secs_f64() * 1000.0;
    let start = Instant::now();
    let doubled = append_slice(src.clone(), dst.clone(), None);
    f.s(2, doubled.clone());
    let append_ms = start.elapsed().as_secs_f64() * 1000.0;
    verify(&src, n); verify(&dst, n); verify(&doubled, 2 * n);
    collect();
    let ss = storage(&src); let ds = storage(&dst); let os = storage(&doubled);
    assert_eq!(ss.1, n); assert_eq!(ds.1, n); assert_eq!(os.1, 2 * n);
    assert_eq!(ss.2, n); assert_eq!(ds.2, n); assert_eq!(os.2, 2 * n);
    if native { assert_eq!(ss.0, 1); assert_eq!(ds.0, 1); assert_eq!(os.0, 1); }
    let retained = ss.0 * ss.2 + ds.0 * ds.2 + os.0 * os.2;
    let stats = heap_stats();
    println!("mode={} payload_mib={} elem_size={} caps={},{},{} retained_element_bytes={} rss_before={} rss_after={} allocate_ms={:.3} copy_16_ms={:.3} copy_mib_per_s={:.2} append_ms={:.3} heap_live={} heap_peak={} collections={} gc_threshold={} allocator=Rust_system", mode, n / 1024 / 1024, ss.0, ss.2, ds.2, os.2, retained, before, rss(), allocate_ms, copy_ms, (16.0 * n as f64 / 1024.0 / 1024.0) / (copy_ms / 1000.0), append_ms, heap_live(), stats.0, stats.1, std::env::var("GOALCHEMY_GC_THRESHOLD").unwrap_or("default_50000".into()));
    black_box(f);
}
