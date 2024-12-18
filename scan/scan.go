package scan

import (
	"runtime"
)

func ScanDirConcurrent(dir string, concurrency int, threshold int64) ([]*FileData, error) {
	CurrentPath.Store("")
	TotalSize.Store(0)
	TotalItems.Store(0)

	root := newRootFileData(dir)
	pool := NewScanPool(concurrency, threshold)
	pool.Start()
	pool.AddTask(root)
	pool.Wait()
	return root.Children, nil
}

func DefaultConcurrency() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}

	return numCPU
}
