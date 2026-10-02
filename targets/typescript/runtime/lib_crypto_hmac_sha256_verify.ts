import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoHmacSha256Verify(t:Task,key:Slice<number>,data:Slice<number>,mac:Slice<number>):void{byteOperation(t,'bool',[key,data,mac],b=>native.hmacVerify(b[0],b[1],b[2]));}
