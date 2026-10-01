//! core.integer.convert: conversion between integer kinds.
use super::*;

pub fn to_i8(a: V) -> V {
    V::Int(w8(a.i()))
}

pub fn to_i16(a: V) -> V {
    V::Int(w16(a.i()))
}

pub fn to_i32(a: V) -> V {
    V::Int(w32(a.i()))
}

pub fn to_i64(a: V) -> V {
    V::Int(w64(a.i()))
}

pub fn to_u8(a: V) -> V {
    V::Int(wu8(a.i()))
}

pub fn to_u16(a: V) -> V {
    V::Int(wu16(a.i()))
}

pub fn to_u32(a: V) -> V {
    V::Int(wu32(a.i()))
}

pub fn to_u64(a: V) -> V {
    V::Int(wu64(a.i()))
}

