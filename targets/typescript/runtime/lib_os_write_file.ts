import { MAX_FILE_BYTES, FileStatus, writeHostFile } from '../types/files.ts';
import type { Slice } from '../types/slice.ts';
export function libOsWriteFile(name:string,data:Slice<number>,perm:number|bigint):bigint{if(data.l>MAX_FILE_BYTES)return BigInt(FileStatus.TooLarge);const b=new Uint8Array(data.l);for(let i=0;i<data.l;i++)b[i]=data.a![data.o+i]!;return BigInt(writeHostFile(name,b,Number(perm)&0o777));}
