import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoHmacSha256(t:Task,key:Slice<number>,data:Slice<number>):void{byteOperation(t,'bytes',[key,data],b=>native.hmac(b[0],b[1]));}
