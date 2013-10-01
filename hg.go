package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type hg struct {
	cmd string
}

func (_ hg) ShortName() string { return "hg" }

var Hg VCS = hg{"hg"}

type hgRepo struct {
	dir string
	hg  *hg
}

func (hg hg) Clone(url, dir string) (Repository, error) {
	r := &hgRepo{dir, &hg}

	cmd := exec.Command("hg", "clone", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("abort: destination '%s' is not empty", dir)) {
			return nil, os.ErrExist
		}
		return nil, fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}

	return r, nil
}

func (hg hg) Open(dir string) (Repository, error) {
	// TODO(sqs): check for .hg or bare repo
	if _, err := os.Stat(dir); err == nil {
		return &hgRepo{dir, &hg}, nil
	} else {
		return nil, err
	}
}

func (hg hg) CloneMirror(url, dir string) error {
	cmd := exec.Command("hg", "clone", "-U", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("abort: destination '%s' is not empty", dir)) {
			return os.ErrExist
		}
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (hg hg) UpdateMirror(dir string) error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *hgRepo) Dir() (dir string) {
	return r.dir
}

func (r *hgRepo) VCS() VCS {
	return r.hg
}

func (r *hgRepo) Download() error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *hgRepo) CheckOut(rev string) (dir string, err error) {
	if rev == "" {
		rev = "default"
	}
	cmd := exec.Command("hg", "update", "-r", rev)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return r.dir, nil
	} else {
		return "", fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
}

func (r *hgRepo) ReadFileAtRevision(path string, rev string) ([]byte, error) {
	cmd := exec.Command("hg", "cat", "-r", rev, "--", path)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return out, nil
	} else {
		if strings.Contains(string(out), fmt.Sprintf("%s: no such file in rev", path)) {
			return nil, os.ErrNotExist
		}
		if strings.Contains(string(out), fmt.Sprintf("abort: unknown revision '%s'!", rev)) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
}
