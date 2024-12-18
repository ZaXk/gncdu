package scan

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ScanPool goroutine pool for scanning tasks
type ScanPool struct {
	concurrency int32          // Number of concurrent scanners
	taskChan    chan *FileData // Task channel
	wg          sync.WaitGroup // Wait group
	closeOnce   sync.Once      // Ensure only close once
	taskCount   int32          // Task counter
	threshold   int64          // Threshold for small files
}

var (
	CurrentPath atomic.Value // Current scanning path
	TotalSize   atomic.Int64 // Current total size
	TotalItems  atomic.Int32 // Current total items
)

// NewScanPool create a new scanning pool
func NewScanPool(concurrency int, threshold int64) *ScanPool {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency()
	}

	return &ScanPool{
		concurrency: int32(concurrency),
		taskChan:    make(chan *FileData, concurrency*4),
		taskCount:   1, // Initial task count is 1 (root directory)
		threshold:   threshold,
	}
}

// Start start the scanning pool
func (p *ScanPool) Start() {
	p.wg.Add(int(p.concurrency))
	for i := int32(0); i < p.concurrency; i++ {
		go p.worker()
	}
}

// worker worker goroutine
func (p *ScanPool) worker() {
	defer p.wg.Done()

	for task := range p.taskChan {
		CurrentPath.Store(task.Path())
		files, err := readDir(task.Path())
		if err != nil {
			atomic.AddInt32(&p.taskCount, -1)
			continue
		}

		var normalFiles []*FileData
		var totalSmallSize int64
		var smallCount int32
		dirCount := int32(0)

		// First pass: classify files
		for _, file := range files {
			TotalItems.Add(1)
			size := file.Size()
			TotalSize.Add(size)

			if !file.IsDir() && size < p.threshold {
				// Small files only accumulate size and count, do not create objects
				totalSmallSize += size
				smallCount++
			} else {
				// Create objects for directories and large files
				child := &FileData{
					Parent: task,
					name:   file.Name(),
					isDir:  file.IsDir(),
					size:   size,
					count:  -1,
					info:   file,
				}
				if child.IsDir() {
					dirCount++
				}
				normalFiles = append(normalFiles, child)
			}
		}

		// If there are small files, create a virtual file to represent them
		if smallCount > 0 {
			virtualFile := &FileData{
				Parent:    task,
				name:      fmt.Sprintf("<Files smaller than %dMB>", p.threshold/MB),
				isDir:     false,
				size:      totalSmallSize,
				count:     smallCount,
				isVirtual: true,
			}
			normalFiles = append(normalFiles, virtualFile)
		}

		// Set the processed file list
		task.SetChildren(normalFiles)

		// Process subdirectories
		if dirCount > 0 {
			atomic.AddInt32(&p.taskCount, dirCount)
			for _, child := range normalFiles {
				if child.IsDir() {
					for i := 0; i < 3; i++ {
						select {
						case p.taskChan <- child:
							goto nextChild
						default:
							time.Sleep(time.Microsecond)
						}
					}
					p.processDir(child)
				nextChild:
				}
			}
		}

		remaining := atomic.AddInt32(&p.taskCount, -1)
		if remaining == 0 {
			p.closeOnce.Do(func() {
				close(p.taskChan)
			})
		}
	}
}

// processDir directly process directories
func (p *ScanPool) processDir(dir *FileData) {
	// Update the current scanning path
	CurrentPath.Store(dir.Path())

	files, err := readDir(dir.Path())
	if err != nil {
		atomic.AddInt32(&p.taskCount, -1)
		return
	}

	var normalFiles []*FileData
	var totalSmallSize int64
	var smallCount int32
	dirCount := int32(0)

	// First pass: classify files
	for _, file := range files {
		TotalItems.Add(1)
		size := file.Size()
		TotalSize.Add(size)

		if !file.IsDir() && size < p.threshold {
			// Small files only accumulate size and count, do not create objects
			totalSmallSize += size
			smallCount++
		} else {
			// Create objects for directories and large files
			child := &FileData{
				Parent: dir,
				name:   file.Name(),
				isDir:  file.IsDir(),
				size:   size,
				count:  -1,
				info:   file,
			}
			if child.IsDir() {
				dirCount++
			}
			normalFiles = append(normalFiles, child)
		}
	}

	// If there are small files, create a virtual file to represent them
	if smallCount > 0 {
		virtualFile := &FileData{
			Parent:    dir,
			name:      fmt.Sprintf("<Files smaller than %dMB>", p.threshold/MB),
			isDir:     false,
			size:      totalSmallSize,
			count:     smallCount,
			isVirtual: true,
		}
		normalFiles = append(normalFiles, virtualFile)
	}

	dir.SetChildren(normalFiles)

	if dirCount > 0 {
		atomic.AddInt32(&p.taskCount, dirCount)
		for _, child := range normalFiles {
			if child.IsDir() {
				p.processDir(child)
			}
		}
	}

	atomic.AddInt32(&p.taskCount, -1)
}

// AddTask add a scanning task
func (p *ScanPool) AddTask(task *FileData) {
	p.taskChan <- task
}

// Wait wait for all tasks to complete
func (p *ScanPool) Wait() {
	p.wg.Wait()
}
