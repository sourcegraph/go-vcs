package vcs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

func (r *hgRepo) CommitLog() ([]*Commit, error) {
	cmd := exec.Command("hg", "log", `--template={node}\n{author|person}\n{author|email}\n{date|rfc3339date}\n\n{desc}\n\x00`)
	cmd.Dir = r.dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	commitEntries := bytes.Split(out, []byte{'\x00'})
	commitEntries = commitEntries[:len(commitEntries)-1] // hg log puts delimiter at end
	commits := make([]*Commit, len(commitEntries))
	for i, e := range commitEntries {
		if len(e) == 0 {
			continue
		}
		commit := new(Commit)
		parts := bytes.SplitN(e, []byte("\n\n"), 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unhandled hg commit entry: %q", string(e))
		}
		header, commitMsg := parts[0], parts[1]

		headers := bytes.Split(header, []byte{'\n'})
		commit.ID = string(headers[0])
		commit.AuthorName = string(headers[1])
		commit.AuthorEmail = string(headers[2])

		var err error
		commit.AuthorDate, err = time.Parse(time.RFC3339, string(headers[3]))
		if err != nil {
			return nil, err
		}

		commit.Message = strings.TrimSpace(string(commitMsg))
		commits[i] = commit
	}

	return commits, nil
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

func (r *hgRepo) CurrentCommitID() (string, error) {
	cmd := exec.Command("hg", "identify", "-i")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
