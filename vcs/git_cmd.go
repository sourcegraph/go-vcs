package vcs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.tools/godoc/vfs"
)

type GitRepositoryCmd struct {
	Dir string
}

// checkSpecArgSafety returns a non-nil err if spec begins with a "-", which could
// cause it to be interpreted as a git command line argument.
func checkSpecArgSafety(spec string) error {
	if strings.HasPrefix(spec, "-") {
		return errors.New("invalid git revision spec (begins with '-')")
	}
	return nil
}

func (r *GitRepositoryCmd) ResolveRevision(spec string) (CommitID, error) {
	if err := checkSpecArgSafety(spec); err != nil {
		return "", err
	}

	cmd := exec.Command("git", "rev-parse", spec)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("unknown revision")) {
			return "", ErrRevisionNotFound
		}
		return "", fmt.Errorf("exec `git rev-parse` failed: %s. Output was:\n\n%s", err, out)
	}
	return CommitID(bytes.TrimSpace(out)), nil
}

func (r *GitRepositoryCmd) ResolveBranch(name string) (CommitID, error) {
	commitID, err := r.ResolveRevision(name)
	if err == ErrRevisionNotFound {
		return "", ErrBranchNotFound
	}
	return commitID, nil
}

func (r *GitRepositoryCmd) ResolveTag(name string) (CommitID, error) {
	commitID, err := r.ResolveRevision(name)
	if err == ErrRevisionNotFound {
		return "", ErrTagNotFound
	}
	return commitID, nil
}

func (r *GitRepositoryCmd) Branches() ([]*Branch, error) {
	refs, err := r.showRef("--heads")
	if err != nil {
		return nil, err
	}

	branches := make([]*Branch, len(refs))
	for i, ref := range refs {
		branches[i] = &Branch{
			Name: strings.TrimPrefix(ref[1], "refs/heads/"),
			Head: CommitID(ref[0]),
		}
	}
	return branches, nil
}

func (r *GitRepositoryCmd) Tags() ([]*Tag, error) {
	refs, err := r.showRef("--tags")
	if err != nil {
		return nil, err
	}

	tags := make([]*Tag, len(refs))
	for i, ref := range refs {
		tags[i] = &Tag{
			Name:     strings.TrimPrefix(ref[1], "refs/tags/"),
			CommitID: CommitID(ref[0]),
		}
	}
	return tags, nil
}

type byteSlices [][]byte

func (p byteSlices) Len() int           { return len(p) }
func (p byteSlices) Less(i, j int) bool { return bytes.Compare(p[i], p[j]) < 0 }
func (p byteSlices) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (r *GitRepositoryCmd) showRef(arg string) ([][2]string, error) {
	cmd := exec.Command("git", "show-ref", arg)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git show-ref ...` failed: %s. Output was:\n\n%s", err, out)
	}

	out = bytes.TrimSuffix(out, []byte("\n")) // remove trailing newline
	lines := bytes.Split(out, []byte("\n"))
	sort.Sort(byteSlices(lines)) // sort for consistency
	refs := make([][2]string, len(lines))
	for i, line := range lines {
		if len(line) <= 41 {
			return nil, errors.New("unexpectedly short (<=41 bytes) line in `git show-ref ...` output")
		}
		id := line[:40]
		name := line[41:]
		refs[i] = [2]string{string(id), string(name)}
	}
	return refs, nil
}

func (r *GitRepositoryCmd) GetCommit(id CommitID) (*Commit, error) {
	if err := checkSpecArgSafety(string(id)); err != nil {
		return nil, err
	}

	commits, _, err := r.commitLog(CommitsOptions{Head: id, N: 1})
	if err != nil {
		return nil, err
	}

	if len(commits) != 1 {
		return nil, fmt.Errorf("git log: expected 1 commit, got %d", len(commits))
	}

	return commits[0], nil
}

func (r *GitRepositoryCmd) Commits(opt CommitsOptions) ([]*Commit, uint, error) {
	if err := checkSpecArgSafety(string(opt.Head)); err != nil {
		return nil, 0, err
	}

	return r.commitLog(opt)
}

func (r *GitRepositoryCmd) commitLog(opt CommitsOptions) ([]*Commit, uint, error) {
	args := []string{"log", `--format=format:%H%x00%aN%x00%aE%x00%at%x00%cN%x00%cE%x00%ct%x00%B%x00%P%x00`}
	if opt.N != 0 {
		args = append(args, "-n", strconv.FormatUint(uint64(opt.N), 10))
	}
	if opt.Skip != 0 {
		args = append(args, "--skip="+strconv.FormatUint(uint64(opt.Skip), 10))
	}
	args = append(args, string(opt.Head))

	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("exec `git log` failed: %s. Output was:\n\n%s", err, out)
	}

	const partsPerCommit = 9 // number of \x00-separated fields per commit
	allParts := bytes.Split(out, []byte{'\x00'})
	numCommits := len(allParts) / partsPerCommit
	commits := make([]*Commit, numCommits)
	for i := 0; i < numCommits; i++ {
		parts := allParts[partsPerCommit*i : partsPerCommit*(i+1)]

		// log outputs are newline separated, so all but the 1st commit ID part
		// has an erroneous leading newline.
		parts[0] = bytes.TrimPrefix(parts[0], []byte{'\n'})

		authorTime, err := strconv.ParseInt(string(parts[3]), 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("parsing git commit author time: %s", err)
		}
		committerTime, err := strconv.ParseInt(string(parts[6]), 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("parsing git commit committer time: %s", err)
		}

		var parents []CommitID
		if parentPart := parts[8]; len(parentPart) > 0 {
			parentIDs := bytes.Split(parentPart, []byte{' '})
			parents = make([]CommitID, len(parentIDs))
			for i, id := range parentIDs {
				parents[i] = CommitID(id)
			}
		}

		commits[i] = &Commit{
			ID:        CommitID(parts[0]),
			Author:    Signature{string(parts[1]), string(parts[2]), time.Unix(authorTime, 0)},
			Committer: &Signature{string(parts[4]), string(parts[5]), time.Unix(committerTime, 0)},
			Message:   string(bytes.TrimSuffix(parts[7], []byte{'\n'})),
			Parents:   parents,
		}
	}

	// Count commits.
	cmd = exec.Command("git", "rev-list", "--count", string(opt.Head))
	cmd.Dir = r.Dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("exec `git rev-list --count` failed: %s. Output was:\n\n%s", err, out)
	}
	out = bytes.TrimSpace(out)
	total, err := strconv.ParseUint(string(out), 10, 64)
	if err != nil {
		return nil, 0, err
	}

	return commits, uint(total), nil
}

func (r *GitRepositoryCmd) Diff(base, head CommitID, opt *DiffOptions) (*Diff, error) {
	if strings.HasPrefix(string(base), "-") || strings.HasPrefix(string(head), "-") {
		// Protect against base or head that is interpreted as command-line option.
		return nil, errors.New("diff revspecs must not start with '-'")
	}

	cmd := exec.Command("git", "diff", "--full-index", string(base), string(head), "--")
	if opt != nil {
		cmd.Args = append(cmd.Args, opt.Paths...)
	}
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git diff` failed: %s. Output was:\n\n%s", err, out)
	}
	return &Diff{
		Raw: string(out),
	}, nil
}

func (r *GitRepositoryCmd) CrossRepoDiff(base CommitID, headRepo Repository, head CommitID, opt *DiffOptions) (*Diff, error) {
	var headDir string // path to head repo on local filesystem
	switch headRepo := headRepo.(type) {
	case *gitRepository:
		headDir = headRepo.dir
	case *GitRepositoryCmd:
		headDir = headRepo.Dir
	default:
		return nil, fmt.Errorf("git cross-repo diff not supported against head repo type %T", headRepo)
	}

	if headDir == r.Dir {
		return r.Diff(base, head, opt)
	}

	// Add remote (if not exists).
	cmd := exec.Command("git", "fetch", headDir, string(head))
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git fetch` failed: %s. Output was:\n\n%s", err, out)
	}

	return r.Diff(base, head, opt)
}

func (r *GitRepositoryCmd) FileSystem(at CommitID) (vfs.FileSystem, error) {
	if err := checkSpecArgSafety(string(at)); err != nil {
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

func (fs *gitFSCmd) Open(name string) (vfs.ReadSeekCloser, error) {
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
	path = filepath.Clean(path)

	cmd := exec.Command("git", "log", "-1", "--format=%ad", string(fs.at),
		"--", path)
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	mtime, err := time.Parse("Mon Jan _2 15:04:05 2006 -0700",
		strings.Trim(string(out), "\n"))
	if err != nil {
		return nil, err
	}

	if path == "." {
		return &fileInfo{mode: os.ModeDir, mtime: mtime}, nil
	}

	// TODO(sqs): follow symlinks (as Stat is required to do)

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
		return &fileInfo{name: filepath.Base(path), mode: os.ModeDir,
			mtime: mtime}, nil
	}

	return &fileInfo{name: filepath.Base(path), size: int64(len(data)),
		mtime: mtime}, nil
}

func (fs *gitFSCmd) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)
	if err := checkSpecArgSafety(path); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "ls-tree", "-z", string(fs.at), path+"/")
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("exists on disk, but not in")) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("exec `git ls-files` failed: %s. Output was:\n\n%s", err, out)
	}

	// in `git show` output for dir, first line is header, 2nd line is blank,
	// and there is a trailing newline.
	lines := bytes.Split(out, []byte{'\x00'})
	fis := make([]os.FileInfo, len(lines)-1)
	for i, line := range lines {
		if i == len(lines)-1 {
			// last entry is empty
			continue
		}

		typ, name := string(line[7:11]), line[53:]

		var mode os.FileMode
		if typ == "tree" {
			mode = os.ModeDir
		} else if typ == "link" {
			mode = os.ModeSymlink
		}

		relName, err := filepath.Rel(path, string(name))
		if err != nil {
			return nil, err
		}
		fis[i] = &fileInfo{name: relName, mode: mode}
	}

	return fis, nil
}

func (fs *gitFSCmd) String() string {
	return fmt.Sprintf("git repository %s commit %s (cmd)", fs.dir, fs.at)
}
