package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/KooQix/term-ai/internal/config"
)

var Logger *log.Logger

func init() {
	configPath, err := config.GetConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(fmt.Sprintf("%s/logs.log", configPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	Logger = log.New(file, "", log.LstdFlags)
}

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
