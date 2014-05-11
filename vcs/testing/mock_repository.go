package testing

import "github.com/sourcegraph/go-vcs/vcs"

type MockRepository struct {
	ResolveRevision_ func(spec string) (vcs.CommitID, error)
	ResolveTag_      func(name string) (vcs.CommitID, error)
	ResolveBranch_   func(name string) (vcs.CommitID, error)

	GetCommit_ func(vcs.CommitID) (*vcs.Commit, error)
	CommitLog_ func(to vcs.CommitID) ([]*vcs.Commit, error)

	FileSystem_ func(at vcs.CommitID) (vcs.FileSystem, error)
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

func (r MockRepository) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
	if r.GetCommit_ == nil {
		return nil, nil
	}
	return r.GetCommit_(id)
}

func (r MockRepository) CommitLog(to vcs.CommitID) ([]*vcs.Commit, error) {
	if r.CommitLog_ == nil {
		return nil, nil
	}
	return r.CommitLog_(to)
}

func (r MockRepository) FileSystem(at vcs.CommitID) (vcs.FileSystem, error) {
	if r.FileSystem_ == nil {
		return nil, nil
	}
	return r.FileSystem_(at)
}
