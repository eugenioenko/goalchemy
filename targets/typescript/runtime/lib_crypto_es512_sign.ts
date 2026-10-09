import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
import { bytes, DeclaredFailure } from '../types/native.ts';
export function libCryptoEs512Sign(t:Task,key:Key|null,data:Slice<number>):void{try{const b=bytes(data);keyOperation(t,'bytes',[key],ls=>native.sign('ES512',ls[0],b));}catch(e){if(!(e instanceof DeclaredFailure))throw e;cryptoOperation(t,'bytes',async()=>null,[],e);}}
