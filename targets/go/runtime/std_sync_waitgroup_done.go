package rt

func StdSyncWaitgroupDone(wg *WaitGroup) { StdSyncWaitgroupAdd(wg, -1) }
