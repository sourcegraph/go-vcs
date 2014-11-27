package git

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	git2go "github.com/libgit2/git2go"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/util"
)

func init() {
	// Overwrite the git opener to return repositories that use the
	// faster libgit2 implementation.
	vcs.RegisterOpener("git", func(dir string) (vcs.Repository, error) {
		return Open(dir)
	})
}

type Repository struct {
	*gitcmd.Repository
	u *git2go.Repository
}

func Open(dir string) (*Repository, error) {
	cr, err := gitcmd.Open(dir)
	if err != nil {
		return nil, err
	}

	u, err := git2go.OpenRepository(dir)
	if err != nil {
		return nil, err
	}
	return &Repository{cr, u}, nil
}

func (r *Repository) ResolveRevision(spec string) (vcs.CommitID, error) {
	o, err := r.u.RevparseSingle(spec)
	if err != nil {
		if err.Error() == fmt.Sprintf("Revspec '%s' not found.", spec) {
			return "", vcs.ErrRevisionNotFound
		}
		return "", err
	}
	defer o.Free()
	return vcs.CommitID(o.Id().String()), nil
}

func (r *Repository) ResolveBranch(name string) (vcs.CommitID, error) {
	b, err := r.u.LookupBranch(name, git2go.BranchLocal)
	if err != nil {
		if err.Error() == fmt.Sprintf("Cannot locate local branch '%s'", name) {
			return "", vcs.ErrBranchNotFound
		}
		return "", err
	}
	return vcs.CommitID(b.Target().String()), nil
}

func (r *Repository) ResolveTag(name string) (vcs.CommitID, error) {
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

	return "", vcs.ErrTagNotFound
}

func (r *Repository) Branches() ([]*vcs.Branch, error) {
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

func (r *Repository) Tags() ([]*vcs.Tag, error) {
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

func (r *Repository) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
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

func (r *Repository) Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, uint, error) {
	walk, err := r.u.Walk()
	if err != nil {
		return nil, 0, err
	}
	defer walk.Free()

	walk.Sorting(git2go.SortTopological)

	oid, err := git2go.NewOid(string(opt.Head))
	if err != nil {
		return nil, 0, err
	}
	if err := walk.Push(oid); err != nil {
		return nil, 0, err
	}

	var commits []*vcs.Commit
	total := uint(0)
	err = walk.Iterate(func(c *git2go.Commit) bool {
		if total >= opt.Skip && (opt.N == 0 || uint(len(commits)) < opt.N) {
			commits = append(commits, r.makeCommit(c))
		}
		total++
		return true
	})
	if err != nil {
		return nil, 0, err
	}

	return commits, total, nil
}

func (r *Repository) makeCommit(c *git2go.Commit) *vcs.Commit {
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

var defaultDiffOptions git2go.DiffOptions

func init() {
	var err error
	defaultDiffOptions, err = git2go.DefaultDiffOptions()
	if err != nil {
		log.Fatalf("Failed to load default git (libgit2/git2go) diff options: %s.", err)
	}
	defaultDiffOptions.IdAbbrev = 40
}

func (r *Repository) Diff(base, head vcs.CommitID, opt *vcs.DiffOptions) (*vcs.Diff, error) {
	gopt := defaultDiffOptions

	baseOID, err := git2go.NewOid(string(base))
	if err != nil {
		return nil, err
	}
	baseCommit, err := r.u.LookupCommit(baseOID)
	if err != nil {
		return nil, err
	}
	baseTree, err := r.u.LookupTree(baseCommit.TreeId())
	if err != nil {
		return nil, err
	}
	defer baseTree.Free()

	headOID, err := git2go.NewOid(string(head))
	if err != nil {
		return nil, err
	}
	headCommit, err := r.u.LookupCommit(headOID)
	if err != nil {
		return nil, err
	}
	headTree, err := r.u.LookupTree(headCommit.TreeId())
	if err != nil {
		return nil, err
	}
	defer headTree.Free()

	if opt != nil {
		if opt.Paths != nil {
			gopt.Pathspec = opt.Paths
		}
	}

	gdiff, err := r.u.DiffTreeToTree(baseTree, headTree, &gopt)
	if err != nil {
		return nil, err
	}
	defer gdiff.Free()

	diff := &vcs.Diff{}

	ndeltas, err := gdiff.NumDeltas()
	if err != nil {
		return nil, err
	}
	for i := 0; i < ndeltas; i++ {
		patch, err := gdiff.Patch(i)
		if err != nil {
			return nil, err
		}
		defer patch.Free()

		patchStr, err := patch.String()
		if err != nil {
			return nil, err
		}

		diff.Raw += patchStr
	}
	return diff, nil
}

func (r *Repository) FileSystem(at vcs.CommitID) (vfs.FileSystem, error) {
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

func (fs *gitFSLibGit2) Open(name string) (vfs.ReadSeekCloser, error) {
	e, err := fs.getEntry(name)
	if err != nil {
		return nil, err
	}

	b, err := fs.repo.LookupBlob(e.Id)
	if err != nil {
		return nil, err
	}
	defer b.Free()

	return util.NopCloser{bytes.NewReader(b.Contents())}, nil
}

func (fs *gitFSLibGit2) Lstat(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)

	mtime, err := fs.getModTime()
	if err != nil {
		return nil, err
	}

	if path == "." {
		return &util.FileInfo{Mode_: os.ModeDir, ModTime_: mtime}, nil
	}

	e, err := fs.getEntry(path)
	if err != nil {
		return nil, err
	}

	fi, err := fs.makeFileInfo(e)
	if err != nil {
		return nil, err
	}
	fi.ModTime_ = mtime

	return fi, nil
}

func (fs *gitFSLibGit2) Stat(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)

	mtime, err := fs.getModTime()
	if err != nil {
		return nil, err
	}

	if path == "." {
		return &util.FileInfo{Mode_: os.ModeDir, ModTime_: mtime}, nil
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
		fi.(*util.FileInfo).Name_ = filepath.Base(path)
		return fi, nil
	}

	fi, err := fs.makeFileInfo(e)
	if err != nil {
		return nil, err
	}
	fi.ModTime_ = mtime

	return fi, nil
}

func (fs *gitFSLibGit2) getModTime() (time.Time, error) {
	commit, err := fs.repo.LookupCommit(fs.oid)
	if err != nil {
		return time.Time{}, err
	}
	return commit.Author().When, nil
}

func (fs *gitFSLibGit2) makeFileInfo(e *git2go.TreeEntry) (*util.FileInfo, error) {
	switch e.Type {
	case git2go.ObjectBlob:
		return fs.fileInfo(e)

	case git2go.ObjectTree:
		return fs.dirInfo(e), nil
	}

	return nil, fmt.Errorf("unexpected object type %v while making file info (expected blob or tree)", e.Type)
}

func (fs *gitFSLibGit2) fileInfo(e *git2go.TreeEntry) (*util.FileInfo, error) {
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

	return &util.FileInfo{
		Name_: e.Name,
		Size_: b.Size(),
		Mode_: mode,
	}, nil
}

func (fs *gitFSLibGit2) dirInfo(e *git2go.TreeEntry) *util.FileInfo {
	return &util.FileInfo{
		Name_: e.Name,
		Mode_: os.ModeDir,
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
			fis[i] = &util.FileInfo{
				Name_: e.Name,
				Mode_: os.ModeSymlink,
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
