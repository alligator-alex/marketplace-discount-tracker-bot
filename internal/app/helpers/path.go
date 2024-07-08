package helpers

import (
	"errors"
	"os"
	"path/filepath"
)

// Return application's root directory.
func GetRootDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(currentDir, ".env")
		if _, err := os.Stat(path); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", errors.New("unable to resolve root directory")
		}
		currentDir = parent
	}

	return currentDir, nil
}
