package vcs

import (
	"fmt"
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

// Open a repository rooted at dir, of vcs type "git" or "hg".
func Open(vcs, dir string) (Repository, error) {
	switch vcs {
	case "git":
		return OpenGitRepository(dir)
	case "hg":
		return OpenHgRepository(dir)
	}
	return nil, fmt.Errorf("unknown VCS type %q", vcs)
}
