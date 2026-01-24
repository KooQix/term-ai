package fileprocessor

import (
	"fmt"
	"os"
	"path/filepath"
)

// scanDirectory scans a directory for supported files
func ScanDirectory(dirPath string) ([]*FileAttachment, error) {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirPath)
	}

	var filePaths []string

	// Walk the directory (only top level by default for safety)
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip subdirectories (only process top level)
			if path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is supported
		if IsSupported(path) {
			filePaths = append(filePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no supported files found in directory")
	}

	// Process all found files
	return ProcessFiles(filePaths)
}
