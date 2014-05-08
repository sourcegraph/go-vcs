package vcs

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"code.google.com/p/go.tools/godoc/vfs"
	"github.com/gogits/git"
)

type LocalGitRepository struct {
	dir string
	u   *git.Repository
}

func OpenLocalGitRepository(dir string) (*LocalGitRepository, error) {
	r, err := git.OpenRepository(dir)
	if err != nil {
		return nil, err
	}
	return &LocalGitRepository{dir, r}, nil
}

func (r *LocalGitRepository) ResolveBranch(name string) (CommitID, error) {
	id, err := r.u.GetCommitIdOfBranch(name)
	return CommitID(id), err
}

func (r *LocalGitRepository) ResolveTag(name string) (CommitID, error) {
	id, err := r.u.GetCommitIdOfTag(name)
	return CommitID(id), err
}

func (r *LocalGitRepository) FileSystem(at CommitID) (vfs.FileSystem, error) {
	c, err := r.u.GetCommit(string(at))
	if err != nil {
		return nil, err
	}
	return &localGitFS{r.dir, &c.Tree, at, r.u}, nil
}

type localGitFS struct {
	dir  string
	tree *git.Tree
	at   CommitID

	repo *git.Repository
}

func (fs *localGitFS) getEntry(path string) (*git.TreeEntry, error) {
	e, err := fs.tree.GetTreeEntryByPath(path)
	if err == git.ErrNotExist {
		c, err := fs.repo.GetCommitOfRelPath(string(fs.at), path)
		if err != nil {
			if err.Error() == "Length must be 40" {
				// this is when `git log` called by GetCommitOfRelPath returns
				// empty (which means the file is not found)
				err = git.ErrNotExist
			}
			return nil, standardizeError(err)
		}
		e, err = c.Tree.GetTreeEntryByPath(path)
	}
	return e, err
}

func (fs *localGitFS) Open(name string) (vfs.ReadSeekCloser, error) {
	e, err := fs.getEntry(name)
	if err != nil {
		return nil, err
	}

	data, err := e.Blob().Data()
	if err != nil {
		return nil, err
	}

	return nopCloser{bytes.NewReader(data)}, nil
}

func (fs *localGitFS) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *localGitFS) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat is required to do)
	return fs.getEntry(path)
}

func (fs *localGitFS) ReadDir(path string) ([]os.FileInfo, error) {
	subtree, err := fs.tree.SubTree(path)
	if err != nil {
		return nil, standardizeError(err)
	}

	entries := subtree.ListEntries()
	fis := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		fis[i] = e
	}
	return fis, nil
}

func (fs *localGitFS) String() string {
	return fmt.Sprintf("local git repository %s commit %s", fs.dir, fs.at)
}

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

func standardizeError(err error) error {
	if err == git.ErrNotExist {
		return os.ErrNotExist
	}
	return err
}
