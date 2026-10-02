import { bytes,byteSlice,b64,unb64,errorBox,DeclaredFailure } from '../types/native.ts';
import type { Slice } from '../types/slice.ts';
export function libEncodingBase64Encode(data:Slice<number>):[any,any]{try{return [b64(bytes(data),false),null];}catch(e){if(!(e instanceof DeclaredFailure))throw e;return ['',errorBox('encoding: invalid input or size')];}}
