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
		return nil, fmt.Errorf("git clone failed: %s\n%s", err, out)
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
		return fmt.Errorf("git fetch --all failed: %s\n%s", err, out)
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
		return "", fmt.Errorf("git checkout %q failed: %s\n%s", rev, err, out)
	}
}
