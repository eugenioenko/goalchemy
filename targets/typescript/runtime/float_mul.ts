import { roundFloat, floatMin, floatMax } from "../types/float.ts";
export function floatMul_f32(a: number, b: number): number { return roundFloat(a * b, 32); }
export function floatMul_f64(a: number, b: number): number { return roundFloat(a * b, 64); }
