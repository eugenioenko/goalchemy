import { roundFloat, floatMin, floatMax } from "../types/float.ts";
export function floatMin_f32(a: number, b: number): number { return roundFloat(floatMin(a,b), 32); }
export function floatMin_f64(a: number, b: number): number { return roundFloat(floatMin(a,b), 64); }
