use super::*;
pub fn float_mul_f32(a: V, b: V) -> V { V::Float(round_float(a.f() * b.f(), 32)) }
pub fn float_mul_f64(a: V, b: V) -> V { V::Float(round_float(a.f() * b.f(), 64)) }
