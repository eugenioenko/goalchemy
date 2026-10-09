import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
import { bytes, DeclaredFailure } from '../types/native.ts';
export function libCryptoRs384Verify(t:Task,key:Key|null,data:Slice<number>,signature:Slice<number>):void{try{const b=bytes(data);const s=bytes(signature);keyOperation(t,'bool',[key],ls=>native.verify('RS384',ls[0],b,s));}catch(e){if(!(e instanceof DeclaredFailure))throw e;cryptoOperation(t,'bool',async()=>null,[],e);}}
