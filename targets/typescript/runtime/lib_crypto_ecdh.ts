import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoEcdh(t:Task,a:Key|null,b:Key|null):void{keyOperation(t,'bytes',[a,b],ls=>native.ecdh(ls[0],ls[1]));}
