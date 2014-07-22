package vcs

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

func getModTime(dir, path string) time.Time {
	stat, err := os.Stat(filepath.Join(dir, path))
	if err != nil {
		return time.Time{}
	}
	return stat.ModTime()
}
