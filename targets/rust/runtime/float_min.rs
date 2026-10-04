use super::*;
pub fn float_min_f32(a: V, b: V) -> V { V::Float(round_float(float_min(a.f(),b.f()), 32)) }
pub fn float_min_f64(a: V, b: V) -> V { V::Float(round_float(float_min(a.f(),b.f()), 64)) }
