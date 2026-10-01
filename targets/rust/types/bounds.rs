//! Go slice-bounds checks and messages. When u is set the bounds have an
//! unsigned 64-bit type: negative values are huge.
use super::*;

fn oob(m: String) -> ! {
    runtime_panic(&format!("slice bounds out of range {}", m))
}

fn st(x: i64, u: bool) -> String {
    if u { (x as u64).to_string() } else { x.to_string() }
}

fn neg(x: i64, u: bool) -> bool {
    !u && x < 0
}

fn gt(x: i64, y: i64, u: bool) -> bool {
    if u { (x as u64) > (y as u64) } else { x > y }
}

pub fn check2(lo: i64, hi: i64, limit: i64, word: &str, u: bool) {
    if neg(hi, u) { oob(format!("[:{}]", st(hi, u))) }
    if gt(hi, limit, u) { oob(format!("[:{}] with {} {}", st(hi, u), word, limit)) }
    if neg(lo, u) { oob(format!("[{}:]", st(lo, u))) }
    if gt(lo, hi, u) { oob(format!("[{}:{}]", st(lo, u), st(hi, u))) }
}

pub fn check3(lo: i64, hi: i64, max: i64, limit: i64, word: &str, u: bool) {
    if neg(max, u) { oob(format!("[::{}]", st(max, u))) }
    if gt(max, limit, u) { oob(format!("[::{}] with {} {}", st(max, u), word, limit)) }
    if neg(hi, u) { oob(format!("[:{}:]", st(hi, u))) }
    if gt(hi, max, u) { oob(format!("[:{}:{}]", st(hi, u), st(max, u))) }
    if neg(lo, u) { oob(format!("[{}::]", st(lo, u))) }
    if gt(lo, hi, u) { oob(format!("[{}:{}:]", st(lo, u), st(hi, u))) }
}
