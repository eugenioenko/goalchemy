import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoAes256GcmEncrypt(t:Task,key:Slice<number>,nonce:Slice<number>,data:Slice<number>,aad:Slice<number>):void{byteOperation(t,'bytes',[key,nonce,data,aad],b=>native.aes(false,b[0],b[1],b[2],b[3]));}
