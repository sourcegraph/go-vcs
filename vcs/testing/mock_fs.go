package testing

import (
	"os"

	godoc_vfs "code.google.com/p/go.tools/godoc/vfs"
	"code.google.com/p/go.tools/godoc/vfs/mapfs"
	"github.com/sourcegraph/go-vcs/vcs"
)

func MapFS(m map[string]string) vcs.FileSystem { return GoDocVFS{mapfs.New(m)} }

// GoDocVFS wraps a godoc/vfs.FileSystem to implement vcs.FileSystem.
type GoDocVFS struct{ godoc_vfs.FileSystem }

// Open implements vcs.FileSystem (using the underlying godoc/vfs.FileSystem
// Open method, which returns an interface with the same methods but at a
// different import path).
func (fs GoDocVFS) Open(name string) (vcs.ReadSeekCloser, error) {
	return fs.FileSystem.Open("/" + name)
}
func (fs GoDocVFS) Lstat(path string) (os.FileInfo, error) { return fs.FileSystem.Lstat("/" + path) }
func (fs GoDocVFS) Stat(path string) (os.FileInfo, error)  { return fs.FileSystem.Stat("/" + path) }
func (fs GoDocVFS) ReadDir(path string) ([]os.FileInfo, error) {
	return fs.FileSystem.ReadDir("/" + path)
}
