//! IEEE binary32 is stored as an exactly widened binary64 value.
use super::*;
pub fn round_float(x: f64, bits: u32) -> f64 { if bits == 32 { (x as f32) as f64 } else { x } }
pub fn integer_float(value: i64, unsigned: bool, bits: u32) -> f64 {
    match (unsigned, bits) {
        (true, 32) => ((value as u64) as f32) as f64,
        (true, _) => (value as u64) as f64,
        (false, 32) => (value as f32) as f64,
        (false, _) => value as f64,
    }
}
pub fn float_integer(x: f64, bits: u32, signed: bool) -> i64 {
    if !signed && bits == 64 {
        return if x.is_finite() && x >= -9223372036854775808.0 && x < 18446744073709551616.0 {
            if x >= 9223372036854775808.0 { ((x - 9223372036854775808.0) as i64) ^ i64::MIN } else { x as i64 }
        } else { i64::MIN };
    }
    let width = if bits <= 16 || (bits == 32 && signed) { 32 } else { 64 };
    let limit = if width == 32 { 2147483648.0 } else { 9223372036854775808.0 };
    let n = if x.is_finite() && x >= -limit && x < limit { x as i64 } else if width == 32 { i32::MIN as i64 } else { i64::MIN };
    if bits == 64 { n } else if signed { n << (64-bits) >> (64-bits) } else { n & ((1i64<<bits)-1) }
}
pub fn float_min(a: f64, b: f64) -> f64 {
    if a.is_nan() || b.is_nan() { f64::NAN } else if a == 0.0 && b == 0.0 { if a.is_sign_negative() || b.is_sign_negative() { -0.0 } else { 0.0 } } else if a < b { a } else { b }
}
pub fn float_max(a: f64, b: f64) -> f64 {
    if a.is_nan() || b.is_nan() { f64::NAN } else if a == 0.0 && b == 0.0 { if a.is_sign_positive() || b.is_sign_positive() { 0.0 } else { -0.0 } } else if a > b { a } else { b }
}
pub fn zero_float() -> V { V::Float(0.0) }

pub fn float_print(x: f64, bits: u32) -> String {
    let x = round_float(x, bits);
    if x.is_nan() { return "NaN".into(); }
    if x.is_infinite() { return if x < 0.0 { "-Inf" } else { "+Inf" }.into(); }
    if x == 0.0 { return if x.is_sign_negative() { "-0" } else { "0" }.into(); }
    let sign = if x < 0.0 { "-" } else { "" };
    let x = x.abs();
    let mut text = String::new();
    for n in 1..=if bits == 32 { 9 } else { 17 } {
        text = format!("{:.*e}", n-1, x);
        if round_float(text.parse::<f64>().unwrap(), bits) == x { break; }
    }
    let (mantissa, exponent) = text.split_once('e').unwrap();
    let digits = mantissa.replace('.', "");
    let digits = digits.trim_end_matches('0');
    let exp: i32 = exponent.parse().unwrap();
    if exp < -4 || exp >= 6 {
        return format!("{}{}{}e{}{:02}", sign, &digits[..1], if digits.len() > 1 { format!(".{}", &digits[1..]) } else { String::new() }, if exp < 0 { "-" } else { "+" }, exp.abs());
    }
    let point = exp + 1;
    format!("{}{}", sign, if point <= 0 { format!("0.{}{}", "0".repeat((-point) as usize), digits) } else if point as usize >= digits.len() { format!("{}{}", digits, "0".repeat(point as usize-digits.len())) } else { format!("{}.{}", &digits[..point as usize], &digits[point as usize..]) })
}
