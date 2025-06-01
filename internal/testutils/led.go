package testutils

import (
	"os"
	"path/filepath"
)

func InitLED(path string) error {
	if err := os.WriteFile(filepath.Join(path, "trigger"), []byte("[none]"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "max_brightness"), []byte("1"), 0644); err != nil {
		return err
	}
	return nil
}
