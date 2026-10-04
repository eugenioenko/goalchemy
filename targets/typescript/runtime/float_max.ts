import { roundFloat, floatMin, floatMax } from "../types/float.ts";
export function floatMax_f32(a: number, b: number): number { return roundFloat(floatMax(a,b), 32); }
export function floatMax_f64(a: number, b: number): number { return roundFloat(floatMax(a,b), 64); }
