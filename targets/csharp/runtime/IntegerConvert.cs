namespace Rt;

/// <summary>core.integer.convert: conversion between integer kinds.</summary>
public static partial class R
{
    public static long to_i8(long a) => Ints.w8(a);
    public static long to_i16(long a) => Ints.w16(a);
    public static long to_i32(long a) => Ints.w32(a);
    public static long to_i64(long a) => a;
    public static long to_u8(long a) => Ints.wu8(a);
    public static long to_u16(long a) => Ints.wu16(a);
    public static long to_u32(long a) => Ints.wu32(a);
    public static long to_u64(long a) => a;
}
