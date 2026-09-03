package xdb

// SetExecutorForTest allows injecting a mock database executor during unit tests.
func (r *DBGeneric[T, PT]) SetExecutorForTest(exec SQLExecutor) {
	r.executor = exec
}
