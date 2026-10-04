import { roundFloat, floatMin, floatMax } from "../types/float.ts";
export function floatNeg_f32(a: number): number { return roundFloat(-a, 32); }
export function floatNeg_f64(a: number): number { return roundFloat(-a, 64); }
