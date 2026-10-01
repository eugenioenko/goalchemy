/* core.integer.convert: conversion between integer kinds. */
#include "gx.h"

gx_V gx_to_i8(gx_V a) { return gx_int(gx_w8(a.u.i)); }
gx_V gx_to_i16(gx_V a) { return gx_int(gx_w16(a.u.i)); }
gx_V gx_to_i32(gx_V a) { return gx_int(gx_w32(a.u.i)); }
gx_V gx_to_i64(gx_V a) { return gx_int(gx_w64(a.u.i)); }
gx_V gx_to_u8(gx_V a) { return gx_int(gx_wu8(a.u.i)); }
gx_V gx_to_u16(gx_V a) { return gx_int(gx_wu16(a.u.i)); }
gx_V gx_to_u32(gx_V a) { return gx_int(gx_wu32(a.u.i)); }
gx_V gx_to_u64(gx_V a) { return gx_int(gx_wu64(a.u.i)); }
