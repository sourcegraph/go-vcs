package vcs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type GitRepositoryCmd struct {
	Dir string
}

// checkSpecArgSafety returns a non-nil err if spec begins with a "-", which could
// cause it to be interpreted as a git command line argument.
func (r *GitRepositoryCmd) checkSpecArgSafety(spec string) error {
	if strings.HasPrefix(spec, "-") {
		return errors.New("invalid git revision spec (begins with '-')")
	}
	return nil
}

func (r *GitRepositoryCmd) ResolveRevision(spec string) (CommitID, error) {
	if err := r.checkSpecArgSafety(spec); err != nil {
		return "", err
	}

	cmd := exec.Command("git", "rev-parse", spec)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec `git rev-parse` failed: %s. Output was:\n\n%s", err, out)
	}
	return CommitID(bytes.TrimSpace(out)), nil
}

func (r *GitRepositoryCmd) ResolveBranch(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *GitRepositoryCmd) ResolveTag(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *GitRepositoryCmd) GetCommit(id CommitID) (*Commit, error) {
	if err := r.checkSpecArgSafety(string(id)); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "log", `--format=format:%H%x00%aN%x00%aE%x00%at%x00%cN%x00%cE%x00%ct%x00%s`, "-n", "1", string(id))
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git log` failed: %s. Output was:\n\n%s", err, out)
	}

	parts := bytes.Split(out, []byte{'\x00'})
	authorTime, err := strconv.ParseInt(string(parts[3]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing git commit author time: %s", err)
	}
	committerTime, err := strconv.ParseInt(string(parts[6]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing git commit committer time: %s", err)
	}
	c := &Commit{
		ID:        CommitID(parts[0]),
		Author:    Signature{string(parts[1]), string(parts[2]), time.Unix(authorTime, 0)},
		Committer: &Signature{string(parts[4]), string(parts[5]), time.Unix(committerTime, 0)},
		Message:   string(parts[7]),
	}

	// get parents
	cmd = exec.Command("git", "log", `--format=format:%H`, string(id))
	cmd.Dir = r.Dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git log` failed: %s. Output was:\n\n%s", err, out)
	}

	lines := bytes.Split(out, []byte{'\n'})
	c.Parents = make([]CommitID, len(lines)-1)
	for i, line := range lines {
		if i == 0 {
			// this commit is not its own parent, so skip it
			continue
		}
		c.Parents[i-1] = CommitID(line)
	}

	return c, nil
}

func (r *GitRepositoryCmd) FileSystem(at CommitID) (FileSystem, error) {
	if err := r.checkSpecArgSafety(string(at)); err != nil {
		return nil, err
	}

	return &gitFSCmd{
		dir: r.Dir,
		at:  at,
	}, nil
}

type gitFSCmd struct {
	dir string
	at  CommitID
}

func (fs *gitFSCmd) Open(name string) (ReadSeekCloser, error) {
	cmd := exec.Command("git", "show", string(fs.at)+":"+name)
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("exists on disk, but not in")) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("exec `git show` failed: %s. Output was:\n\n%s", err, out)
	}
	return nopCloser{bytes.NewReader(out)}, nil
}

func (fs *gitFSCmd) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *gitFSCmd) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat is required to do)

	path = filepath.Clean(path)

	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(data, []byte(fmt.Sprintf("tree %s:%s\n", fs.at, path))) {
		// dir
		return &fileInfo{name: filepath.Base(path), mode: os.ModeDir}, nil
	}

	return &fileInfo{name: filepath.Base(path), size: int64(len(data))}, nil
}

func (fs *gitFSCmd) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)

	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// in `git show` output for dir, first line is header, 2nd line is blank,
	// and there is a trailing newline.
	lines := bytes.Split(data, []byte{'\n'})
	fis := make([]os.FileInfo, len(lines)-3)
	for i, line := range lines {
		if i < 2 || i == len(lines)-1 {
			continue
		}
		fis[i-2] = &fileInfo{name: string(line)}
	}

	return fis, nil
}

func (fs *gitFSCmd) String() string {
	return fmt.Sprintf("git repository %s commit %s (cmd)", fs.dir, fs.at)
}
