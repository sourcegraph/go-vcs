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
	url, dir string
	git      *git
}

func (git git) Clone(url, dir string) (Repository, error) {
	r := &gitRepo{url, dir, &git}

	cmd := exec.Command("git", "clone", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("fatal: destination path '%s' already exists", dir)) {
			return nil, os.ErrExist
		}
		return nil, fmt.Errorf("git clone failed: %s\n%s", err, out)
	}

	return r, nil
}

func (r *gitRepo) Dir() (dir string) {
	return r.dir
}

func (r *gitRepo) VCS() VCS {
	return r.git
}

func (r *gitRepo) Download() error {
	panic("not implemented")
}

func (r *gitRepo) CheckOut(rev string) (dir string, err error) {
	cmd := exec.Command("git", "checkout", rev)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return r.dir, nil
	} else {
		return "", fmt.Errorf("git checkout %q failed: %s\n%s", rev, err, out)
	}
}
