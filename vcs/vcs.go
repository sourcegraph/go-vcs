package vcs

import (
	"os"
	"path/filepath"
	"time"
)

func getModTime(dir, path string) time.Time {
	stat, err := os.Stat(filepath.Join(dir, path))
	if err != nil {
		return time.Time{}
	}
	return stat.ModTime()
}
