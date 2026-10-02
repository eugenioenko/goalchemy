import { type Task } from './task_spawn.ts';
import type { Slice } from '../types/slice.ts';
import * as native from '../types/crypto.ts';
import { cryptoOperation, keyOperation, byteOperation, type Key } from './lib_crypto_close.ts';
export function libCryptoPublicPem(t:Task,key:Key|null):void{keyOperation(t,'string',[key],ls=>native.publicPEM(ls[0]));}
