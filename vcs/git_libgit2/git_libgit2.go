package git_libgit2

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	git2go "github.com/libgit2/git2go"
	"github.com/sourcegraph/go-vcs/vcs"
)

type GitRepositoryLibGit2 struct {
	Dir string
	u   *git2go.Repository
}

func OpenGitRepositoryLibGit2(dir string) (*GitRepositoryLibGit2, error) {
	r, err := git2go.OpenRepository(dir)
	if err != nil {
		return nil, err
	}
	return &GitRepositoryLibGit2{dir, r}, nil
}

func init() {
	vcs.OpenGitRepository = OpenGitRepository
}

type gitRepository struct {
	dir string
	*GitRepositoryLibGit2
	cmd *vcs.GitRepositoryCmd
}

func (r *gitRepository) MirrorUpdate() error {
	return vcs.GitMirrorUpdate(r.dir)
}

// OpenGitRepository opens dir (the root directory of a Git repository) using
// libgit2.
func OpenGitRepository(dir string) (vcs.GitRepository, error) {
	native, err := OpenGitRepositoryLibGit2(dir)
	if err != nil {
		return nil, err
	}

	return &gitRepository{dir, native, &vcs.GitRepositoryCmd{dir}}, nil
}

func (r *GitRepositoryLibGit2) ResolveRevision(spec string) (vcs.CommitID, error) {
	o, err := r.u.RevparseSingle(spec)
	if err != nil {
		return "", err
	}
	defer o.Free()
	return vcs.CommitID(o.Id().String()), nil
}

func (r *GitRepositoryLibGit2) ResolveBranch(name string) (vcs.CommitID, error) {
	b, err := r.u.LookupBranch(name, git2go.BranchLocal)
	if err != nil {
		return "", err
	}
	return vcs.CommitID(b.Target().String()), nil
}

func (r *GitRepositoryLibGit2) ResolveTag(name string) (vcs.CommitID, error) {
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
			return vcs.CommitID(ref.Target().String()), nil
		}
	}

	return "", git2go.MakeGitError(git2go.ErrClassTag)
}

func (r *GitRepositoryLibGit2) Branches() ([]*vcs.Branch, error) {
	refs, err := r.u.NewReferenceIterator()
	if err != nil {
		return nil, err
	}

	var bs []*vcs.Branch
	for {
		ref, err := refs.Next()
		if isErrIterOver(err) {
			break
		}
		if err != nil {
			return nil, err
		}
		if ref.IsBranch() {
			bs = append(bs, &vcs.Branch{Name: ref.Shorthand(), Head: vcs.CommitID(ref.Target().String())})
		}
	}

	sort.Sort(vcs.Branches(bs))
	return bs, nil
}

func (r *GitRepositoryLibGit2) Tags() ([]*vcs.Tag, error) {
	refs, err := r.u.NewReferenceIterator()
	if err != nil {
		return nil, err
	}

	var ts []*vcs.Tag
	for {
		ref, err := refs.Next()
		if isErrIterOver(err) {
			break
		}
		if err != nil {
			return nil, err
		}
		if ref.IsTag() {
			ts = append(ts, &vcs.Tag{Name: ref.Shorthand(), CommitID: vcs.CommitID(ref.Target().String())})
		}
	}

	sort.Sort(vcs.Tags(ts))
	return ts, nil
}

func (r *GitRepositoryLibGit2) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
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

func (r *GitRepositoryLibGit2) CommitLog(to vcs.CommitID) ([]*vcs.Commit, error) {
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

	var commits []*vcs.Commit
	err = walk.Iterate(func(c *git2go.Commit) bool {
		commits = append(commits, r.makeCommit(c))
		return true
	})
	if err != nil {
		return nil, err
	}

	return commits, nil
}

func (r *GitRepositoryLibGit2) makeCommit(c *git2go.Commit) *vcs.Commit {
	var parents []vcs.CommitID
	if pc := c.ParentCount(); pc > 0 {
		parents = make([]vcs.CommitID, pc)
		for i := 0; i < int(pc); i++ {
			parents[i] = vcs.CommitID(c.ParentId(uint(i)).String())
		}
	}

	au, cm := c.Author(), c.Committer()
	return &vcs.Commit{
		ID:        vcs.CommitID(c.Id().String()),
		Author:    vcs.Signature{au.Name, au.Email, au.When},
		Committer: &vcs.Signature{cm.Name, cm.Email, cm.When},
		Message:   strings.TrimSuffix(c.Message(), "\n"),
		Parents:   parents,
	}
}

func (r *GitRepositoryLibGit2) FileSystem(at vcs.CommitID) (vcs.FileSystem, error) {
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

	return &gitFSLibGit2{r.Dir, oid, at, tree, r.u}, nil
}

type gitFSLibGit2 struct {
	dir  string
	oid  *git2go.Oid
	at   vcs.CommitID
	tree *git2go.Tree

	repo *git2go.Repository
}

func (fs *gitFSLibGit2) getEntry(path string) (*git2go.TreeEntry, error) {
	path = filepath.Clean(path)
	e, err := fs.tree.EntryByPath(path)
	if err != nil {
		return nil, standardizeLibGit2Error(err)
	}

	return e, nil
}

func (fs *gitFSLibGit2) Open(name string) (vcs.ReadSeekCloser, error) {
	e, err := fs.getEntry(name)
	if err != nil {
		return nil, err
	}

	b, err := fs.repo.LookupBlob(e.Id)
	if err != nil {
		return nil, err
	}
	defer b.Free()

	return nopCloser{bytes.NewReader(b.Contents())}, nil
}

func (fs *gitFSLibGit2) Lstat(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)

	mtime, err := fs.getModTime()
	if err != nil {
		return nil, err
	}

	if path == "." {
		return &fileInfo{mode: os.ModeDir, mtime: mtime}, nil
	}

	e, err := fs.getEntry(path)
	if err != nil {
		return nil, err
	}

	fi, err := fs.makeFileInfo(e)
	if err != nil {
		return nil, err
	}
	fi.(*fileInfo).mtime = mtime

	return fi, nil
}

func (fs *gitFSLibGit2) Stat(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)

	mtime, err := fs.getModTime()
	if err != nil {
		return nil, err
	}

	if path == "." {
		return &fileInfo{mode: os.ModeDir, mtime: mtime}, nil
	}

	e, err := fs.getEntry(path)
	if err != nil {
		return nil, err
	}

	if e.Filemode == git2go.FilemodeLink {
		// dereference symlink
		b, err := fs.repo.LookupBlob(e.Id)
		if err != nil {
			return nil, err
		}

		derefPath := string(b.Contents())
		fi, err := fs.Lstat(derefPath)
		if err != nil {
			return nil, err
		}
		fi.(*fileInfo).name = filepath.Base(path)
		return fi, nil
	}

	fi, err := fs.makeFileInfo(e)
	if err != nil {
		return nil, err
	}
	fi.(*fileInfo).mtime = mtime

	return fi, nil
}

func (fs *gitFSLibGit2) getModTime() (time.Time, error) {
	commit, err := fs.repo.LookupCommit(fs.oid)
	if err != nil {
		return time.Time{}, err
	}
	return commit.Author().When, nil
}

func (fs *gitFSLibGit2) makeFileInfo(e *git2go.TreeEntry) (os.FileInfo, error) {
	switch e.Type {
	case git2go.ObjectBlob:
		return fs.fileInfo(e)

	case git2go.ObjectTree:
		return fs.dirInfo(e), nil
	}

	return nil, fmt.Errorf("unexpected object type %v while making file info (expected blob or tree)", e.Type)
}

func (fs *gitFSLibGit2) fileInfo(e *git2go.TreeEntry) (os.FileInfo, error) {
	b, err := fs.repo.LookupBlob(e.Id)
	if err != nil {
		return nil, err
	}
	defer b.Free()

	var mode os.FileMode
	if e.Filemode == git2go.FilemodeBlobExecutable {
		mode |= 0111
	}
	if e.Filemode == git2go.FilemodeLink {
		mode |= os.ModeSymlink
	}

	return &fileInfo{
		name: e.Name,
		size: b.Size(),
		mode: mode,
	}, nil
}

func (fs *gitFSLibGit2) dirInfo(e *git2go.TreeEntry) os.FileInfo {
	return &fileInfo{
		name: e.Name,
		mode: os.ModeDir,
	}
}

func (fs *gitFSLibGit2) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)

	var subtree *git2go.Tree
	if path == "." {
		subtree = fs.tree
	} else {
		e, err := fs.getEntry(path)
		if err != nil {
			return nil, err
		}

		subtree, err = fs.repo.LookupTree(e.Id)
		if err != nil {
			return nil, err
		}
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
		case git2go.ObjectCommit:
			// git submodule
			// TODO(sqs): somehow encode that this is a git submodule and not
			// just a symlink (which is a hack)
			fis[i] = &fileInfo{
				name: e.Name,
				mode: os.ModeSymlink,
			}
		default:
			return nil, fmt.Errorf("unexpected object type %v while reading dir (expected blob or tree)", e.Type)
		}
	}

	return fis, nil
}

func (fs *gitFSLibGit2) String() string {
	return fmt.Sprintf("git repository %s commit %s (libgit2)", fs.dir, fs.at)
}

func isErrIterOver(err error) bool {
	if e, ok := err.(*git2go.GitError); ok {
		return e != nil && e.Code == git2go.ErrIterOver
	}
	return false
}

func standardizeLibGit2Error(err error) error {
	if err != nil && strings.Contains(err.Error(), "does not exist in the given tree") {
		return os.ErrNotExist
	}
	return err
}

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

type fileInfo struct {
	name  string
	mode  os.FileMode
	size  int64
	mtime time.Time
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.mtime }
func (fi *fileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi *fileInfo) Sys() interface{}   { return nil }
