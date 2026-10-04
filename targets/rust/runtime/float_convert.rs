use super::*;
pub fn float_convert(x: V, bits: u32) -> V { V::Float(round_float(x.f(),bits)) }
pub fn integer_float_convert(x: V, unsigned: bool, bits: u32) -> V { V::Float(integer_float(x.i(),unsigned,bits)) }
pub fn float_integer_convert(x: V, bits: u32, signed: bool) -> V { V::Int(float_integer(x.f(),bits,signed)) }
