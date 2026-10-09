import { logEnabled,logLevelNumber } from '../types/log.ts';
export function libLogEnabled(level:bigint):boolean{return logEnabled(logLevelNumber(level));}
