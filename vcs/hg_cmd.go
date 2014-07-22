package vcs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type HgRepositoryCmd struct {
	Dir string
}

func (r *HgRepositoryCmd) ResolveRevision(spec string) (CommitID, error) {
	cmd := exec.Command("hg", "identify", "--debug", "-i", "--rev="+spec)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec `hg identify` failed: %s. Output was:\n\n%s", err, out)
	}
	return CommitID(bytes.TrimSpace(out)), nil
}

func (r *HgRepositoryCmd) ResolveTag(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *HgRepositoryCmd) ResolveBranch(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *HgRepositoryCmd) GetCommit(id CommitID) (*Commit, error) {
	commits, err := r.commitLog(string(id))
	if err != nil {
		return nil, err
	}

	if len(commits) != 1 {
		return nil, fmt.Errorf("hg log: expected 1 commit, got %d", len(commits))
	}

	return commits[0], nil
}

func (r *HgRepositoryCmd) CommitLog(to CommitID) ([]*Commit, error) {
	return r.commitLog(string(to) + ":0")
}

var hgNullParentNodeID = []byte("0000000000000000000000000000000000000000")

func (r *HgRepositoryCmd) commitLog(revSpec string) ([]*Commit, error) {
	cmd := exec.Command("hg", "log", `--template={node}\x00{author|person}\x00{author|email}\x00{date|rfc3339date}\x00{desc}\x00{p1node}\x00{p2node}\x00`, "--rev="+revSpec)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg log` failed: %s. Output was:\n\n%s", err, out)
	}

	const partsPerCommit = 7 // number of \x00-separated fields per commit
	allParts := bytes.Split(out, []byte{'\x00'})
	numCommits := len(allParts) / partsPerCommit
	commits := make([]*Commit, numCommits)
	for i := 0; i < numCommits; i++ {
		parts := allParts[partsPerCommit*i : partsPerCommit*(i+1)]
		id := CommitID(parts[0])

		authorTime, err := time.Parse(time.RFC3339, string(parts[3]))
		if err != nil {
			return nil, err
		}

		parents, err := r.getParents(id)
		if err != nil {
			return nil, fmt.Errorf("r.GetParents failed: %s. Output was:\n\n%s", err, out)
		}

		commits[i] = &Commit{
			ID:      id,
			Author:  Signature{string(parts[1]), string(parts[2]), authorTime},
			Message: string(parts[4]),
			Parents: parents,
		}
	}
	return commits, nil
}

func (r *HgRepositoryCmd) getParents(revSpec CommitID) ([]CommitID, error) {
	var parents []CommitID

	cmd := exec.Command("hg", "parents", "-r", string(revSpec), "--template",
		`{node}\x00{author|person}\x00{author|email}\x00{date|rfc3339date}\x00{desc}\x00{p1node}\x00{p2node}\x00`)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg parents` failed: %s. Output was:\n\n%s", err, out)
	}

	const partsPerCommit = 7 // number of \x00-separated fields per commit
	allParts := bytes.Split(out, []byte{'\x00'})
	numCommits := len(allParts) / partsPerCommit
	for i := 0; i < numCommits; i++ {
		parts := allParts[partsPerCommit*i : partsPerCommit*(i+1)]

		if p1 := parts[0]; len(p1) > 0 && !bytes.Equal(p1, hgNullParentNodeID) {
			parents = append(parents, CommitID(p1))
		}
		if p2 := parts[5]; len(p2) > 0 && !bytes.Equal(p2, hgNullParentNodeID) {
			parents = append(parents, CommitID(p2))
		}
		if p3 := parts[6]; len(p3) > 0 && !bytes.Equal(p3, hgNullParentNodeID) {
			parents = append(parents, CommitID(p3))
		}
	}

	return parents, nil
}

func (r *HgRepositoryCmd) FileSystem(at CommitID) (FileSystem, error) {
	return &hgFSCmd{
		dir: r.Dir,
		at:  at,
	}, nil
}

type hgFSCmd struct {
	dir string
	at  CommitID
}

func (fs *hgFSCmd) Open(name string) (ReadSeekCloser, error) {
	cmd := exec.Command("hg", "cat", "--rev="+string(fs.at), "--", name)
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("no such file in rev")) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("exec `hg cat` failed: %s. Output was:\n\n%s", err, out)
	}
	return nopCloser{bytes.NewReader(out)}, nil
}

func (fs *hgFSCmd) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *hgFSCmd) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat is required to do)

	var mtime time.Time

	cmd := exec.Command("hg", "log", "-l1", `--template={date|date}`,
		"-r "+string(fs.at)+":0", "--", path)
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	mtime, err = time.Parse("Mon Jan 02 15:04:05 2006 +0000",
		strings.Trim(string(out), "\n"))
	if err != nil {
		return nil, err
	}

	// this just determines if the file exists.
	cmd = exec.Command("hg", "locate", "--rev="+string(fs.at), "--", path)
	cmd.Dir = fs.dir
	err = cmd.Run()
	if err != nil {
		// hg doesn't track dirs, so use a workaround to see if path is a dir.
		if _, err := fs.ReadDir(path); err == nil {
			return &fileInfo{name: filepath.Base(path), mode: os.ModeDir,
				mtime: mtime}, nil
		}
		return nil, os.ErrNotExist
	}

	// read file to determine file size
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)

	return &fileInfo{name: filepath.Base(path), size: int64(len(data)),
		mtime: mtime}, nil
}

func (fs *hgFSCmd) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)
	// This combination of --include and --exclude opts gets all the files in
	// the dir specified by path, plus all files one level deeper (but no
	// deeper). This lets us list the files *and* subdirs in the dir without
	// needlessly listing recursively.
	cmd := exec.Command("hg", "locate", "--rev="+string(fs.at), "--include="+path, "--exclude="+filepath.Clean(path)+"/*/*/*")
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg cat` failed: %s. Output was:\n\n%s", err, out)
	}

	subdirs := make(map[string]struct{})
	prefix := []byte(path + "/")
	files := bytes.Split(out, []byte{'\n'})
	var fis []os.FileInfo
	for _, nameb := range files {
		nameb = bytes.TrimPrefix(nameb, prefix)
		if len(nameb) == 0 {
			continue
		}
		if bytes.Contains(nameb, []byte{'/'}) {
			subdir := strings.SplitN(string(nameb), "/", 2)[0]
			if _, seen := subdirs[subdir]; !seen {
				fis = append(fis, &fileInfo{name: subdir, mode: os.ModeDir})
				subdirs[subdir] = struct{}{}
			}
			continue
		}
		fis = append(fis, &fileInfo{name: filepath.Base(string(nameb))})
	}

	return fis, nil
}

func (fs *hgFSCmd) String() string {
	return fmt.Sprintf("hg repository %s commit %s (cmd)", fs.dir, fs.at)
}
