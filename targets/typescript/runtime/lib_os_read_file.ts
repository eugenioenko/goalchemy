import { readHostFile } from '../types/files.ts';
import { byteSlice } from '../types/native.ts';
import type { Slice } from '../types/slice.ts';
export function libOsReadFile(name:string):[Slice<number>,bigint]{const r=readHostFile(name);return typeof r==='number'?[byteSlice(null),BigInt(r)]:[byteSlice(r),0n];}
