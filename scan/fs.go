package scan

import (
	"os"
)

// readDir reads all files and subdirectories in the specified directory
// returns a list of file information and possible errors
func readDir(path string) ([]os.FileInfo, error) {
	// Open directory
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	// Read directory contents
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	return files, nil
}
