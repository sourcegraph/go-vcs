package vcs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Repository interface {
	ResolveRevision(spec string) (CommitID, error)
	ResolveTag(name string) (CommitID, error)
	ResolveBranch(name string) (CommitID, error)

	GetCommit(CommitID) (*Commit, error)
	CommitLog(to CommitID) ([]*Commit, error)

	FileSystem(at CommitID) (FileSystem, error)
}

var (
	ErrBranchNotFound   = errors.New("branch not found")
	ErrCommitNotFound   = errors.New("commit not found")
	ErrRevisionNotFound = errors.New("revision not found")
	ErrTagNotFound      = errors.New("tag not found")
)

type CommitID string

type Commit struct {
	ID        CommitID
	Author    Signature
	Committer *Signature `json:",omitempty"`
	Message   string
	Parents   []CommitID `json:",omitempty"`
}

type Signature struct {
	Name  string
	Email string
	Date  time.Time
}

type FileSystem interface {
	Open(name string) (ReadSeekCloser, error)
	Lstat(path string) (os.FileInfo, error)
	Stat(path string) (os.FileInfo, error)
	ReadDir(path string) ([]os.FileInfo, error)
	String() string
}

// A ReadSeekCloser can Read, Seek, and Close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// Open a repository rooted at dir, of vcs type "git" or "hg".
func Open(vcs, dir string) (Repository, error) {
	switch vcs {
	case "git":
		return OpenGitRepository(dir)
	case "hg":
		return OpenHgRepository(dir)
	}
	return nil, fmt.Errorf("unknown VCS type %q", vcs)
}

func Clone(vcs, url, dir string) (Repository, error) {
	switch vcs {
	case "git":
		return CloneGitRepository(url, dir)
	case "hg":
		return CloneHgRepository(url, dir)
	}
	return nil, fmt.Errorf("unknown VCS type %q", vcs)
}

// MirrorRepository provides the MirrorUpdate, in addition to all Repository
// methods. See OpenMirror for more information about mirror repositories.
type MirrorRepository interface {
	Repository

	// MirrorUpdate mirror updates all branches, tags, etc., to match the origin
	// repository of the mirror.
	MirrorUpdate() error
}

// OpenMirror opens the repository rooted at dir (with vcs type "git" or "hg")
// as a mirror. It is assumed that repositories opened with OpenMirror were
// previously created with CloneMirror or as described below; otherwise, the
// behavior is undefined.
//
// The definition of mirror repositories is as follows:
//
// * Git: cloned with `git clone --mirror` (implies bare)
// * Hg: cloned with `hg pull -U` (bare)
//
// The MirrorRepository interface exposes an additional method, MirrorUpdate,
// that updates all branches, tags, etc., to match the origin repository.
//
// The mirror-related functionality in package vcs is provided as a convenience
// because mirroring repositories is a use case that's anticipated to be common.
func OpenMirror(vcs, dir string) (MirrorRepository, error) {
	r, err := Open(vcs, dir)
	if err != nil {
		return nil, err
	}

	return r.(MirrorRepository), nil
}

func CloneMirror(vcs, url, dir string) (MirrorRepository, error) {
	switch vcs {
	case "git":
		cmd := exec.Command("git", "clone", "--mirror", "--", url, dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("exec `git clone --mirror` failed: %s. Output was:\n\n%s", err, out)
		}
	case "hg":
		cmd := exec.Command("hg", "clone", "-U", "--", url, dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("exec `hg clone -U` failed: %s. Output was:\n\n%s", err, out)
		}
	default:
		return nil, fmt.Errorf("unknown VCS type %q", vcs)
	}
	return OpenMirror(vcs, dir)
}
