import { roundFloat, floatMin, floatMax } from "../types/float.ts";
export function floatSub_f32(a: number, b: number): number { return roundFloat(a - b, 32); }
export function floatSub_f64(a: number, b: number): number { return roundFloat(a - b, 64); }
