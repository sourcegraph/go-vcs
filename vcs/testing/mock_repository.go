package testing

import (
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

type MockRepository struct {
	ResolveRevision_ func(spec string) (vcs.CommitID, error)
	ResolveTag_      func(name string) (vcs.CommitID, error)
	ResolveBranch_   func(name string) (vcs.CommitID, error)

	Branches_ func() ([]*vcs.Branch, error)
	Tags_     func() ([]*vcs.Tag, error)

	GetCommit_ func(vcs.CommitID) (*vcs.Commit, error)
	Commits_   func(vcs.CommitsOptions) ([]*vcs.Commit, uint, error)

	BlameFile_ func(path string, opt *vcs.BlameOptions) ([]*vcs.Hunk, error)

	FileSystem_ func(at vcs.CommitID) (vfs.FileSystem, error)
}

var (
	_ interface {
		vcs.Repository
		vcs.Blamer
	} = MockRepository{}
)

func (r MockRepository) ResolveRevision(spec string) (vcs.CommitID, error) {
	return r.ResolveRevision_(spec)
}

func (r MockRepository) ResolveTag(name string) (vcs.CommitID, error) {
	return r.ResolveTag_(name)
}

func (r MockRepository) ResolveBranch(name string) (vcs.CommitID, error) {
	return r.ResolveBranch_(name)
}

func (r MockRepository) Branches() ([]*vcs.Branch, error) {
	return r.Branches_()
}

func (r MockRepository) Tags() ([]*vcs.Tag, error) {
	return r.Tags_()
}

func (r MockRepository) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
	return r.GetCommit_(id)
}

func (r MockRepository) Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, uint, error) {
	return r.Commits_(opt)
}

func (r MockRepository) BlameFile(path string, opt *vcs.BlameOptions) ([]*vcs.Hunk, error) {
	return r.BlameFile_(path, opt)
}

func (r MockRepository) FileSystem(at vcs.CommitID) (vfs.FileSystem, error) {
	return r.FileSystem_(at)
}
