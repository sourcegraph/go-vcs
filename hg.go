package vcs

import (
	"fmt"
	"os/exec"
)

type hg struct {
	cmd string
}

var Hg VCS = hg{"hg"}

type hgRepo struct {
	url, dir string
	hg       *hg
}

func (hg hg) Clone(url, dir string) (Repository, error) {
	r := &hgRepo{url, dir, &hg}

	cmd := exec.Command("hg", "clone", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("hg clone failed: %s\n%s", err, out)
	}

	return r, nil
}

func (r *hgRepo) Dir() (dir string) {
	return r.dir
}

func (r *hgRepo) VCS() VCS {
	return r.hg
}

func (r *hgRepo) Download() error {
	panic("not implemented")
}

func (r *hgRepo) CheckOut(rev string) (dir string, err error) {
	cmd := exec.Command("hg", "update", "-r", rev)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return r.dir, nil
	} else {
		return "", fmt.Errorf("hg update -r %q failed: %s\n%s", rev, err, out)
	}
}
