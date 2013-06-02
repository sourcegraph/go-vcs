package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func assertFileContains(t *testing.T, dir, filename, want string) {
	path := filepath.Join(dir, filename)
	if data, err := ioutil.ReadFile(path); err == nil {
		if got := string(data); want != got {
			t.Fatalf("file %q: want %q, got %q", path, want, got)
		}
	} else {
		t.Fatalf("ReadFile %q: %s", path, err)
	}
}

func assertNotFileExists(t *testing.T, dir, filename string) {
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file %q exists, but want it to not exist", path)
	}
}
