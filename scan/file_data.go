package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileData struct {
	Parent    *FileData
	name      string
	isDir     bool
	size      int64
	Children  []*FileData
	count     int32
	info      os.FileInfo
	mu        sync.Mutex
	infoOnce  sync.Once
	isVirtual bool
}

func newRootFileData(dir string) *FileData {
	// Get actual information of root directory
	info, err := os.Stat(dir)
	if err != nil {
		// If failed to get info, use default values
		return &FileData{
			name:  dir,
			isDir: true,
			size:  0,
			count: 0,
		}
	}

	return &FileData{
		name:  dir,
		isDir: true,
		size:  info.Size(),
		count: 0,
		info:  info,
	}
}

func newFileData(parent *FileData, name string, isDir bool, size int64) *FileData {
	return &FileData{
		Parent: parent,
		name:   name,
		isDir:  isDir,
		size:   size,
		count:  -1,
	}
}

func (d FileData) Root() bool {
	return d.info == nil
}

func (d FileData) Label() string {
	// First check if it's a virtual file
	if d.isVirtual {
		return d.name
	}

	// Then check if it's root directory
	if d.Root() {
		return "/.."
	}

	// Finally check if it's a directory
	if d.isDir {
		return d.name + "/"
	}

	return d.name
}

func (d *FileData) Path() string {
	if d.Parent == nil {
		return d.name
	}
	return filepath.Join(d.Parent.Path(), d.name)
}

func (d FileData) String() string {
	return d.Path()
}

func (d *FileData) Count() int {
	d.mu.Lock()
	if int(d.count) != -1 {
		count := int(d.count)
		d.mu.Unlock()
		return count
	}
	d.mu.Unlock()

	// If it's a virtual file, return its represented file count
	if d.isVirtual {
		return int(d.count)
	}

	total := 0

	for _, child := range d.Children {
		if child.isVirtual {
			// Virtual file, add its represented file count
			total += int(child.count)
		} else if child.IsDir() {
			// Directory itself counts as 1
			total += 1
			// Add count of directory contents
			child.mu.Lock()
			childCount := child.count
			child.mu.Unlock()

			if childCount == -1 {
				total += child.Count()
			} else {
				total += int(childCount)
			}
		} else {
			// Regular file counts as 1
			total += 1
		}
	}

	d.mu.Lock()
	d.count = int32(total)
	d.mu.Unlock()

	return total
}

func (d *FileData) Size() int64 {
	d.mu.Lock()
	if d.size != -1 {
		size := d.size
		d.mu.Unlock()
		return size
	}
	d.mu.Unlock()

	var total int64
	if !d.IsDir() {
		if d.info != nil {
			total = d.info.Size()
		}
		d.mu.Lock()
		d.size = total
		d.mu.Unlock()
		return total
	}

	for _, child := range d.Children {
		child.mu.Lock()
		childSize := child.size
		child.mu.Unlock()

		if childSize == -1 {
			total += child.Size()
		} else {
			total += childSize
		}
	}

	d.mu.Lock()
	d.size = total
	d.mu.Unlock()

	return total
}

func (d *FileData) SetChildren(children []*FileData) {
	d.mu.Lock()
	// Set new children
	d.Children = children
	// Reset statistics
	d.size = -1
	d.count = -1
	d.mu.Unlock()

	// Asynchronously update parent node statistics
	if d.Parent != nil {
		go func() {
			d.Parent.mu.Lock()
			d.Parent.size = -1
			d.Parent.count = -1
			d.Parent.mu.Unlock()
		}()
	}
}

func (d *FileData) Delete() error {
	return os.RemoveAll(d.Path())
}

func hasDir(files []os.FileInfo) bool {
	for _, file := range files {
		if file.IsDir() {
			return true
		}
	}
	return false
}

func (d *FileData) Name() string {
	return d.name
}

func (d *FileData) IsDir() bool {
	return d.isDir
}

func (d *FileData) Info() os.FileInfo {
	d.infoOnce.Do(func() {
		info, err := os.Lstat(d.Path())
		if err == nil {
			d.info = info
		}
	})
	return d.info
}

func GroupSmallFiles(files []*FileData, threshold int64) {
	for _, dir := range files {
		if dir.IsDir() {
			// Recursively process subdirectories
			GroupSmallFiles(dir.Children, threshold)
		}
	}

	// Group files in current directory
	var smallFiles []*FileData
	var remainFiles []*FileData
	var totalSmallSize int64
	smallCount := int32(0)

	for _, f := range files {
		if !f.IsDir() && f.Size() < threshold {
			smallFiles = append(smallFiles, f)
			totalSmallSize += f.Size()
			smallCount++
		} else {
			remainFiles = append(remainFiles, f)
		}
	}

	// If there are small files, create a virtual file
	if len(smallFiles) > 0 {
		virtualFile := &FileData{
			Parent:    smallFiles[0].Parent, // Use first file's parent directory
			name:      fmt.Sprintf("<Files smaller than %dMB>", threshold/MB),
			isDir:     false,
			size:      totalSmallSize,
			count:     smallCount,
			isVirtual: true,
		}
		remainFiles = append(remainFiles, virtualFile)
	}

	// Update original file list
	if len(smallFiles) > 0 && len(files) > 0 {
		// Ensure files[0].Parent exists
		if files[0].Parent != nil {
			files[0].Parent.SetChildren(remainFiles)
		}
	}
}
