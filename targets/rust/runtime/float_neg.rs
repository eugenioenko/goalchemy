use super::*;
pub fn float_neg_f32(a: V) -> V { V::Float(round_float(-a.f(), 32)) }
pub fn float_neg_f64(a: V) -> V { V::Float(round_float(-a.f(), 64)) }
