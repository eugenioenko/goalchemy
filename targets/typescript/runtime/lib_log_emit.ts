import { logEmit,logLevelNumber } from '../types/log.ts';
import { strings } from '../types/native.ts';
import type { Slice } from '../types/slice.ts';
export function libLogEmit(level:bigint,unixNano:bigint,message:string,attrs:Slice<string>,text:string):void{logEmit(logLevelNumber(level),unixNano,message,strings(attrs),text);}
