use super::*;
pub fn float_max_f32(a: V, b: V) -> V { V::Float(round_float(float_max(a.f(),b.f()), 32)) }
pub fn float_max_f64(a: V, b: V) -> V { V::Float(round_float(float_max(a.f(),b.f()), 64)) }
