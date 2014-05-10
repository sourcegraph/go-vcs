package vcs

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	git2go "github.com/libgit2/git2go"
)

type GitRepositoryLibGit2 struct {
	dir string
	u   *git2go.Repository
}

func OpenGitRepositoryLibGit2(dir string) (*GitRepositoryLibGit2, error) {
	r, err := git2go.OpenRepository(dir)
	if err != nil {
		return nil, err
	}
	return &GitRepositoryLibGit2{dir, r}, nil
}

func (r *GitRepositoryLibGit2) ResolveRevision(spec string) (CommitID, error) {
	o, err := r.u.RevparseSingle(spec)
	if err != nil {
		return "", err
	}
	defer o.Free()
	return CommitID(o.Id().String()), nil
}

func (r *GitRepositoryLibGit2) ResolveBranch(name string) (CommitID, error) {
	b, err := r.u.LookupBranch(name, git2go.BranchLocal)
	if err != nil {
		return "", err
	}
	return CommitID(b.Target().String()), nil
}

func (r *GitRepositoryLibGit2) ResolveTag(name string) (CommitID, error) {
	// TODO(sqs): slow way to iterate through tags because git_tag_lookup is not
	// in git2go yet
	refs, err := r.u.NewReferenceIterator()
	if err != nil {
		return "", err
	}

	for {
		ref, err := refs.Next()
		if err != nil {
			break
		}
		if ref.IsTag() && ref.Shorthand() == name {
			return CommitID(ref.Target().String()), nil
		}
	}

	return "", git2go.MakeGitError(git2go.ErrClassTag)
}

func (r *GitRepositoryLibGit2) GetCommit(id CommitID) (*Commit, error) {
	oid, err := git2go.NewOid(string(id))
	if err != nil {
		return nil, err
	}

	c, err := r.u.LookupCommit(oid)
	if err != nil {
		return nil, err
	}
	defer c.Free()

	return r.makeCommit(c), nil
}

func (r *GitRepositoryLibGit2) CommitLog(to CommitID) ([]*Commit, error) {
	oid, err := git2go.NewOid(string(to))
	if err != nil {
		return nil, err
	}

	walk, err := r.u.Walk()
	if err != nil {
		return nil, err
	}
	defer walk.Free()

	err = walk.Push(oid)
	if err != nil {
		return nil, err
	}

	var commits []*Commit
	err = walk.Iterate(func(c *git2go.Commit) bool {
		commits = append(commits, r.makeCommit(c))
		return true
	})
	if err != nil {
		return nil, err
	}

	return commits, nil
}

func (r *GitRepositoryLibGit2) makeCommit(c *git2go.Commit) *Commit {
	var parents []CommitID
	if pc := c.ParentCount(); pc > 0 {
		parents = make([]CommitID, pc)
		for i := 0; i < int(pc); i++ {
			parents[i] = CommitID(c.ParentId(uint(i)).String())
		}
	}

	au, cm := c.Author(), c.Committer()
	return &Commit{
		ID:        CommitID(c.Id().String()),
		Author:    Signature{au.Name, au.Email, au.When},
		Committer: &Signature{cm.Name, cm.Email, cm.When},
		Message:   strings.TrimSuffix(c.Message(), "\n"),
		Parents:   parents,
	}
}

func (r *GitRepositoryLibGit2) FileSystem(at CommitID) (FileSystem, error) {
	oid, err := git2go.NewOid(string(at))
	if err != nil {
		return nil, err
	}

	c, err := r.u.LookupCommit(oid)
	if err != nil {
		return nil, err
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	return &gitFSLibGit2{r.dir, oid, at, tree, r.u}, nil
}

type gitFSLibGit2 struct {
	dir  string
	oid  *git2go.Oid
	at   CommitID
	tree *git2go.Tree

	repo *git2go.Repository
}

func (fs *gitFSLibGit2) Open(name string) (ReadSeekCloser, error) {
	e, err := fs.tree.EntryByPath(name)
	if err != nil {
		return nil, standardizeLibGit2Error(err)
	}

	b, err := fs.repo.LookupBlob(e.Id)
	if err != nil {
		return nil, err
	}
	defer b.Free()

	return nopCloser{bytes.NewReader(b.Contents())}, nil
}

func (fs *gitFSLibGit2) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *gitFSLibGit2) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat is required to do)

	e, err := fs.tree.EntryByPath(path)
	if err != nil {
		return nil, standardizeLibGit2Error(err)
	}

	switch e.Type {
	case git2go.ObjectBlob:
		return fs.fileInfo(e)

	case git2go.ObjectTree:
		return fs.dirInfo(e), nil
	}

	return nil, fmt.Errorf("unexpected object type %v", e.Type)
}

func (fs *gitFSLibGit2) fileInfo(e *git2go.TreeEntry) (os.FileInfo, error) {
	b, err := fs.repo.LookupBlob(e.Id)
	if err != nil {
		return nil, err
	}
	defer b.Free()

	return &fileInfo{
		name: e.Name,
		size: b.Size(),
		mode: os.FileMode(e.Filemode),
	}, nil
}

func (fs *gitFSLibGit2) dirInfo(e *git2go.TreeEntry) os.FileInfo {
	return &fileInfo{
		name: e.Name,
		mode: os.ModeDir,
	}
}

func (fs *gitFSLibGit2) ReadDir(path string) ([]os.FileInfo, error) {
	e, err := fs.tree.EntryByPath(path)
	if err != nil {
		return nil, standardizeLibGit2Error(err)
	}

	subtree, err := fs.repo.LookupTree(e.Id)
	if err != nil {
		return nil, err
	}

	fis := make([]os.FileInfo, int(subtree.EntryCount()))
	for i := uint64(0); i < subtree.EntryCount(); i++ {
		e := subtree.EntryByIndex(i)

		switch e.Type {
		case git2go.ObjectBlob:
			fi, err := fs.fileInfo(e)
			if err != nil {
				return nil, err
			}
			fis[i] = fi
		case git2go.ObjectTree:
			fis[i] = fs.dirInfo(e)
		default:
			return nil, fmt.Errorf("unexpected object type %v", e.Type)
		}
	}

	return fis, nil
}

func (fs *gitFSLibGit2) String() string {
	return fmt.Sprintf("git repository %s commit %s (libgit2)", fs.dir, fs.at)
}

func standardizeLibGit2Error(err error) error {
	if err != nil && strings.Contains(err.Error(), "does not exist in the given tree") {
		return os.ErrNotExist
	}
	return err
}
