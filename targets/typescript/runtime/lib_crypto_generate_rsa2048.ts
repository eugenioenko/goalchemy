import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoGenerateRsa2048(t:Task):void{cryptoOperation(t,'key',()=>native.generate('RSA'));}
