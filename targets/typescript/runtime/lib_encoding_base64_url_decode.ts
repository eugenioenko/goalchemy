import { bytes,byteSlice,b64,unb64,errorBox,DeclaredFailure } from '../types/native.ts';
import type { Slice } from '../types/slice.ts';
export function libEncodingBase64UrlDecode(data:string):[any,any]{try{return [byteSlice(unb64(data,true)),null];}catch(e){if(!(e instanceof DeclaredFailure))throw e;return [byteSlice(null),errorBox('encoding: invalid input or size')];}}
