package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// Get the absolute path of a given directory, and ensure the path exists
func GetAbsolutePath(dirPath string) (string, error) {
	absPath := dirPath
	if !filepath.IsAbs(dirPath) {
		var err error

		absPath, err = filepath.Abs(dirPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	// Now, ensure the path exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory does not exist: %w", err)
		}
		return "", fmt.Errorf("failed to stat directory: %w", err)
	}

	return absPath, nil
}
