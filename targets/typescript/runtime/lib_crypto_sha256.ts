import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoSha256(t:Task,data:Slice<number>):void{byteOperation(t,'bytes',[data],b=>native.digest(b[0]));}
