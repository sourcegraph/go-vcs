package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type git struct {
	cmd string
}

func (_ git) ShortName() string { return "git" }

var Git VCS = git{"git"}

type gitRepo struct {
	dir string
	git *git
}

func (git git) Clone(url, dir string) (Repository, error) {
	r := &gitRepo{dir, &git}

	cmd := exec.Command("git", "clone", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("fatal: destination path '%s' already exists", dir)) {
			return nil, os.ErrExist
		}
		return nil, fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}

	return r, nil
}

func (git git) Open(dir string) (Repository, error) {
	// TODO(sqs): check for .git or bare repo
	if _, err := os.Stat(dir); err == nil {
		return &gitRepo{dir, &git}, nil
	} else {
		return nil, err
	}
}

func (git git) CloneMirror(url, dir string) error {
	cmd := exec.Command("git", "clone", "--mirror", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("fatal: destination path '%s' already exists", dir)) {
			return os.ErrExist
		}
		return fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (git git) UpdateMirror(dir string) error {
	cmd := exec.Command("git", "remote", "update")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *gitRepo) Dir() (dir string) {
	return r.dir
}

func (r *gitRepo) VCS() VCS {
	return r.git
}

func (r *gitRepo) Download() error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *gitRepo) CheckOut(rev string) (dir string, err error) {
	if rev == "" {
		rev = "master"
	}
	cmd := exec.Command("git", "checkout", rev)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return r.dir, nil
	} else {
		return "", fmt.Errorf("git %v failed: %s\n%s", cmd.Args, rev, err, out)
	}
}

func (r *gitRepo) ReadFileAtRevision(path string, rev string) ([]byte, error) {
	cmd := exec.Command("git", "show", rev+":"+path)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return out, nil
	} else {
		if strings.Contains(string(out), fmt.Sprintf("fatal: Path '%s' does not exist", path)) {
			return nil, os.ErrNotExist
		}
		if strings.Contains(string(out), fmt.Sprintf("fatal: Invalid object name '%s'", rev)) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}
}
