package vcs

import (
	"bytes"
	"fmt"
	"net/mail"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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

func (r *gitRepo) CommitLog() ([]*Commit, error) {
	cmd := exec.Command("git", "log", "-s", "-z", "--use-mailmap", "--pretty=raw")
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
	commits := make([]*Commit, len(commitEntries))
	for i, e := range commitEntries {
		commit := new(Commit)
		parts := bytes.SplitN(e, []byte("\n\n"), 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unhandled commit entry: %q", string(e))
		}
		headers, commitMsg := parts[0], parts[1]

		// Parse commit header.
		for _, header := range bytes.Split(headers, []byte{'\n'}) {
			parts := bytes.SplitN(header, []byte{' '}, 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("unhandled header: %q", string(header))
			}
			what, data := parts[0], parts[1]
			switch string(what) {
			case "commit":
				commit.ID = string(data)
			case "author":
				// format: "Author Name <email@example.com> 1387407774 -0800"
				parts := bytes.Split(data, []byte{' '})
				// last 2 are date
				if len(parts) <= 2 {
					return nil, fmt.Errorf("unhandled 'author' header line: %q", string(header))
				}
				authorstr := bytes.Join(parts[:len(parts)-2], []byte{' '})
				addr, err := mail.ParseAddress(string(authorstr))

				if err != nil {
					return nil, err
				}
				commit.AuthorName = addr.Name
				commit.AuthorEmail = addr.Address

				epochSecStr, tzOffsetStr := parts[len(parts)-2], parts[len(parts)-1]
				epochSec, err := strconv.Atoi(string(epochSecStr))
				if err != nil {
					return nil, err
				}
				tzOffset, err := strconv.Atoi(string(tzOffsetStr))
				if err != nil {
					return nil, err
				}
				tzOffsetHours := float64(tzOffset) / 100.0
				tzOffsetMins := time.Duration(tzOffsetHours * 60.0)
				commit.AuthorDate = time.Unix(int64(epochSec), 0).Add(time.Minute * tzOffsetMins)
			}
		}

		commit.Message = strings.TrimSpace(string(commitMsg))

		commits[i] = commit
	}

	return commits, nil
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

func (r *gitRepo) ReadFileAtRevision(path string, rev string) (content []byte, filetype FileType, err error) {
	cmd := exec.Command("git", "show", rev+":"+path)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		re, _ := regexp.Compile("^tree .*:" + path)
		if re.MatchString(string(out)) { // dir
			files := strings.Split(string(out), "\n")
			files = files[2 : len(files)-1]
			sort.Sort(fileSlice(files))
			filelist := strings.Join(files, "\n")
			return []byte(filelist), Dir, nil
		} else { // file
			return out, File, nil
		}
	} else {
		if strings.Contains(string(out), fmt.Sprintf("fatal: Path '%s' does not exist", path)) {
			return nil, File, os.ErrNotExist
		}
		if strings.Contains(string(out), fmt.Sprintf("fatal: Invalid object name '%s'", rev)) {
			return nil, File, os.ErrNotExist
		}
		return nil, File, fmt.Errorf("git %v failed: %s\n%s", cmd.Args, err, out)
	}
}

func (r *gitRepo) CurrentCommitID() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type fileSlice []string

func (s fileSlice) Len() int {
	return len(s)
}
func (s fileSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s fileSlice) Less(i, j int) bool {
	if strings.HasSuffix(s[i], "/") && !strings.HasSuffix(s[j], "/") {
		return true
	} else if !strings.HasSuffix(s[i], "/") && strings.HasSuffix(s[j], "/") {
		return false
	} else {
		return s[i] < s[j]
	}
}
