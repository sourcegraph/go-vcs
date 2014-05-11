package vcs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogits/git"
)

type GitRepositoryNative struct {
	dir string
	u   *git.Repository
}

func OpenGitRepositoryNative(dir string) (*GitRepositoryNative, error) {
	r, err := git.OpenRepository(dir)
	if err != nil {
		return nil, err
	}
	return &GitRepositoryNative{dir, r}, nil
}

func (r *GitRepositoryNative) ResolveRevision(spec string) (CommitID, error) {
	id, _ := r.ResolveBranch(spec)
	if id != "" {
		return id, nil
	}
	return r.ResolveTag(spec)
}

func (r *GitRepositoryNative) ResolveBranch(name string) (CommitID, error) {
	id, err := r.u.GetCommitIdOfBranch(name)
	return CommitID(id), err
}

func (r *GitRepositoryNative) ResolveTag(name string) (CommitID, error) {
	id, err := r.u.GetCommitIdOfTag(name)
	return CommitID(id), err
}

func (r *GitRepositoryNative) GetCommit(id CommitID) (*Commit, error) {
	c, err := r.u.GetCommit(string(id))
	if err != nil {
		return nil, err
	}

	return r.makeCommit(c)
}

func (r *GitRepositoryNative) CommitLog(to CommitID) ([]*Commit, error) {
	c, err := r.u.GetCommit(string(to))
	if err != nil {
		return nil, err
	}

	cs, err := c.CommitsBefore()
	if err != nil {
		return nil, err
	}

	commits := make([]*Commit, cs.Len())
	for i, c := 0, cs.Front(); c != nil; c = c.Next() {
		commits[i], err = r.makeCommit(c.Value.(*git.Commit))
		if err != nil {
			return nil, err
		}

		i++
	}
	return commits, nil
}

func (r *GitRepositoryNative) makeCommit(c *git.Commit) (*Commit, error) {
	var parents []CommitID
	if pc := c.ParentCount(); pc > 0 {
		parents = make([]CommitID, pc)
		for i := 0; i < pc; i++ {
			pid, err := c.ParentId(i)
			if err != nil {
				return nil, err
			}
			parents[i] = CommitID(pid.String())
		}
	}

	return &Commit{
		ID:        CommitID(c.Id.String()),
		Author:    Signature{c.Author.Name, c.Author.Email, c.Author.When},
		Committer: &Signature{c.Committer.Name, c.Committer.Email, c.Committer.When},
		Message:   strings.TrimSuffix(c.CommitMessage, "\n"),
		Parents:   parents,
	}, nil
}

func (r *GitRepositoryNative) FileSystem(at CommitID) (FileSystem, error) {
	c, err := r.u.GetCommit(string(at))
	if err != nil {
		return nil, err
	}
	return &gitFSNative{r.dir, &c.Tree, at, r.u}, nil
}

type gitFSNative struct {
	dir  string
	tree *git.Tree
	at   CommitID

	repo *git.Repository
}

func (fs *gitFSNative) getEntry(path string) (*git.TreeEntry, error) {
	e, err := fs.tree.GetTreeEntryByPath(path)
	if err == git.ErrNotExist {
		c, err := fs.repo.GetCommitOfRelPath(string(fs.at), path)
		if err != nil {
			if err.Error() == "Length must be 40" {
				// this is when `git log` called by GetCommitOfRelPath returns
				// empty (which means the file is not found)
				err = git.ErrNotExist
			}
			return nil, standardizeGitError(err)
		}
		e, err = c.Tree.GetTreeEntryByPath(path)
	}
	return e, err
}

func (fs *gitFSNative) Open(name string) (ReadSeekCloser, error) {
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

func (fs *gitFSNative) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *gitFSNative) Stat(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)

	if path == "." {
		return &fileInfo{mode: os.ModeDir}, nil
	}

	// TODO(sqs): follow symlinks (as Stat is required to do)
	return fs.getEntry(path)
}

func (fs *gitFSNative) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)

	var subtree *git.Tree
	var err error
	if path == "." {
		subtree = fs.tree
	} else {
		subtree, err = fs.tree.SubTree(path)
		if err != nil {
			return nil, standardizeGitError(err)
		}
	}

	entries := subtree.ListEntries()
	fis := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		fis[i] = e
	}
	return fis, nil
}

func (fs *gitFSNative) String() string {
	return fmt.Sprintf("git repository %s commit %s (native)", fs.dir, fs.at)
}

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

func standardizeGitError(err error) error {
	if err == git.ErrNotExist {
		return os.ErrNotExist
	}
	return err
}
