package vcs

import "code.google.com/p/go.tools/godoc/vfs"

type Repository interface {
	ResolveBranch(name string) (CommitID, error)
	ResolveTag(name string) (CommitID, error)

	FileSystem(at CommitID) (vfs.FileSystem, error)
}

type CommitID string

type Commit struct {
	ID CommitID
}
