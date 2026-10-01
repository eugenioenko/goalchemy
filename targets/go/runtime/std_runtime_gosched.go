package rt

// StdRuntimeGosched is a pause primitive that requeues the running task.
func StdRuntimeGosched(t *Task) { yieldTask(t) }
