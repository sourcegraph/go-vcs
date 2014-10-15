package testing

import (
	"code.google.com/p/go.tools/godoc/vfs"
	"github.com/sourcegraph/go-vcs/vcs"
)

type MockRepository struct {
	ResolveRevision_ func(spec string) (vcs.CommitID, error)
	ResolveTag_      func(name string) (vcs.CommitID, error)
	ResolveBranch_   func(name string) (vcs.CommitID, error)

	Branches_ func() ([]*vcs.Branch, error)
	Tags_     func() ([]*vcs.Tag, error)

	GetCommit_ func(vcs.CommitID) (*vcs.Commit, error)
	Commits_   func(vcs.CommitsOptions) ([]*vcs.Commit, uint, error)

	FileSystem_ func(at vcs.CommitID) (vfs.FileSystem, error)
}

var _ vcs.Repository = MockRepository{}

func (r MockRepository) ResolveRevision(spec string) (vcs.CommitID, error) {
	if r.ResolveRevision_ == nil {
		return "", nil
	}
	return r.ResolveRevision_(spec)
}

func (r MockRepository) ResolveTag(name string) (vcs.CommitID, error) {
	if r.ResolveTag_ == nil {
		return "", nil
	}
	return r.ResolveTag_(name)
}

func (r MockRepository) ResolveBranch(name string) (vcs.CommitID, error) {
	if r.ResolveBranch_ == nil {
		return "", nil
	}
	return r.ResolveBranch_(name)
}

func (r MockRepository) Branches() ([]*vcs.Branch, error) {
	if r.Branches_ == nil {
		return nil, nil
	}
	return r.Branches_()
}

func (r MockRepository) Tags() ([]*vcs.Tag, error) {
	if r.Tags_ == nil {
		return nil, nil
	}
	return r.Tags_()
}

func (r MockRepository) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
	if r.GetCommit_ == nil {
		return nil, nil
	}
	return r.GetCommit_(id)
}

func (r MockRepository) Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, uint, error) {
	if r.Commits_ == nil {
		return nil, 0, nil
	}
	return r.Commits_(opt)
}

func (r MockRepository) FileSystem(at vcs.CommitID) (vfs.FileSystem, error) {
	if r.FileSystem_ == nil {
		return nil, nil
	}
	return r.FileSystem_(at)
}
