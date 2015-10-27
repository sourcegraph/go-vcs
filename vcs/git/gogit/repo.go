package git

import (
	"fmt"

	"github.com/gogits/git"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func init() {
	// Overwrite the git opener to return repositories that use the
	// gogits native-go implementation.
	vcs.RegisterOpener("git", func(dir string) (vcs.Repository, error) {
		return Open(dir)
	})
}

// Repository is a git VCS repository.
type Repository struct {
	repo *git.Repository
}

func (r *Repository) String() string {
	return fmt.Sprintf("git (gogit) repo at %s", r.repo.Path)
}

func Open(dir string) (*Repository, error) {
	repo, err := git.OpenRepository(dir)
	if err != nil {
		// FIXME: Wrap in vcs error?
		return nil, err
	}

	return &Repository{
		repo: repo,
	}, nil
}

// ResolveRevision returns the revision that the given revision
// specifier resolves to, or a non-nil error if there is no such
// revision.
func (r *Repository) ResolveRevision(spec string) (vcs.CommitID, error) {
	panic("gogit: not implemented")
}

// ResolveTag returns the tag with the given name, or
// ErrTagNotFound if no such tag exists.
func (r *Repository) ResolveTag(name string) (vcs.CommitID, error) {
	panic("gogit: not implemented")
}

// ResolveBranch returns the branch with the given name, or
// ErrBranchNotFound if no such branch exists.
func (r *Repository) ResolveBranch(name string) (vcs.CommitID, error) {
	panic("gogit: not implemented")
}

// Branches returns a list of all branches in the repository.
func (r *Repository) Branches(branchesOpts vcs.BranchesOptions) ([]*vcs.Branch, error) {
	panic("gogit: not implemented")
}

// Tags returns a list of all tags in the repository.
func (r *Repository) Tags() ([]*vcs.Tag, error) {
	panic("gogit: not implemented")
}

// GetCommit returns the commit with the given commit ID, or
// ErrCommitNotFound if no such commit exists.
func (r *Repository) GetCommit(commitID vcs.CommitID) (*vcs.Commit, error) {
	panic("gogit: not implemented")
}

// Commits returns all commits matching the options, as well as
// the total number of commits (the count of which is not subject
// to the N/Skip options).
//
// Optionally, the caller can request the total not to be computed,
// as this can be expensive for large branches.
func (r *Repository) Commits(commitOpts vcs.CommitsOptions) (commits []*vcs.Commit, total uint, err error) {
	panic("gogit: not implemented")
}

// Committers returns the per-author commit statistics of the repo.
func (r *Repository) Committers(committerOpts vcs.CommittersOptions) ([]*vcs.Committer, error) {
	panic("gogit: not implemented")
}

// FileSystem opens the repository file tree at a given commit ID.
func (r *Repository) FileSystem(at vcs.CommitID) (vfs.FileSystem, error) {
	// Implementations may choose to check that the commit exists
	// before FileSystem returns or to defer the check until
	// operations are performed on the filesystem. (For example, an
	// implementation proxying a remote filesystem may not want to
	// incur the round-trip to check that the commit exists.)
	panic("gogit: not implemented")
}
