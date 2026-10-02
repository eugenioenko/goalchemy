import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoHkdfSha256(t:Task,secret:Slice<number>,salt:Slice<number>,info:Slice<number>,n:bigint):void{byteOperation(t,'bytes',[secret,salt,info],b=>native.hkdf(b[0],b[1],b[2],n));}
