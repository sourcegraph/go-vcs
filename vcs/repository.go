package vcs

import (
	"io"
	"os"
)

type Repository interface {
	ResolveRevision(spec string) (CommitID, error)
	ResolveTag(name string) (CommitID, error)

	FileSystem(at CommitID) (FileSystem, error)
}

type CommitID string

type Commit struct {
	ID CommitID
}

type FileSystem interface {
	Open(name string) (ReadSeekCloser, error)
	Lstat(path string) (os.FileInfo, error)
	Stat(path string) (os.FileInfo, error)
	ReadDir(path string) ([]os.FileInfo, error)
	String() string
}

// A ReadSeekCloser can Read, Seek, and Close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}
