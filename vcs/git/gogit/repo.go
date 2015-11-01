package git

import (
	"fmt"
	"strings"

	"github.com/gogits/git"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sqs/pbtypes"
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

	// TODO: Do we need locking?
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
	id, err := r.repo.GetCommitIdOfTag(name)
	if err != nil {
		return "", vcs.ErrTagNotFound
	}
	return vcs.CommitID(id), nil
}

// ResolveBranch returns the branch with the given name, or
// ErrBranchNotFound if no such branch exists.
func (r *Repository) ResolveBranch(name string) (vcs.CommitID, error) {
	id, err := r.repo.GetCommitIdOfBranch(name)
	if err != nil {
		return "", vcs.ErrBranchNotFound
	}
	return vcs.CommitID(id), nil
}

// Branches returns a list of all branches in the repository.
func (r *Repository) Branches(opt vcs.BranchesOptions) ([]*vcs.Branch, error) {
	names, err := r.repo.GetBranches()
	if err != nil {
		return nil, err
	}
	defaultOpt := vcs.BranchesOptions{}
	if opt != defaultOpt {
		return nil, fmt.Errorf("vcs.BranchesOptions options not implemented")
	}

	var branches []*vcs.Branch
	for _, name := range names {
		id, err := r.ResolveBranch(name)
		if err != nil {
			return nil, err
		}
		branch := &vcs.Branch{Name: name, Head: id}
		branches = append(branches, branch)
	}
	return branches, nil
}

// Tags returns a list of all tags in the repository.
func (r *Repository) Tags() ([]*vcs.Tag, error) {
	names, err := r.repo.GetTags()
	if err != nil {
		return nil, err
	}

	tags := make([]*vcs.Tag, 0, len(names))
	for _, name := range names {
		id, err := r.ResolveTag(name)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &vcs.Tag{Name: name, CommitID: vcs.CommitID(id)})
	}
	return tags, nil
}

// GetCommit returns the commit with the given commit ID, or
// ErrCommitNotFound if no such commit exists.
func (r *Repository) GetCommit(commitID vcs.CommitID) (*vcs.Commit, error) {
	commit, err := r.repo.GetCommit(string(commitID))
	if err != nil {
		// FIXME: Check error to make sure it's actually not found and not a different failure.
		//        Unfortunately it's not a fixed error var: https://github.com/gogits/git/issues/13
		return nil, vcs.ErrCommitNotFound
	}

	var committer *vcs.Signature
	if commit.Committer != nil {
		committer = &vcs.Signature{
			Name:  commit.Committer.Name,
			Email: commit.Committer.Email,
			Date:  pbtypes.NewTimestamp(commit.Committer.When),
		}
	}

	n := commit.ParentCount()
	parents := make([]vcs.CommitID, 0, n)
	for i := 0; i < commit.ParentCount(); i++ {
		id, err := commit.ParentId(i)
		if err != nil {
			return nil, err
		}
		parents = append(parents, vcs.CommitID(id.String()))
	}

	return &vcs.Commit{
		ID: vcs.CommitID(commit.Id.String()),
		// TODO: Check nil on commit.Author?
		Author: vcs.Signature{
			Name:  commit.Author.Name,
			Email: commit.Author.Email,
			Date:  pbtypes.NewTimestamp(commit.Author.When),
		},
		Committer: committer,
		Message:   strings.TrimSuffix(commit.Message(), "\n"),
		Parents:   parents,
	}, nil
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
