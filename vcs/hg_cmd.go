package vcs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

func (r *HgRepositoryCmd) GetCommit(id CommitID) (*Commit, error) {
	cmd := exec.Command("hg", "log", "--limit", "1", `--template={node}\x00{author|person}\x00{author|email}\x00{date|rfc3339date}\x00{desc}`, "--rev="+string(id))
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg log` failed: %s. Output was:\n\n%s", err, out)
	}

	parts := bytes.Split(out, []byte{'\x00'})
	authorTime, err := time.Parse(time.RFC3339, string(parts[3]))
	if err != nil {
		return nil, err
	}
	c := &Commit{
		ID:      CommitID(parts[0]),
		Author:  Signature{string(parts[1]), string(parts[2]), authorTime},
		Message: string(parts[4]),
	}

	// get parents
	cmd = exec.Command("hg", "log", `--template={node}\n`, "--rev="+string(id)+":0")
	cmd.Dir = r.Dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg log` failed: %s. Output was:\n\n%s", err, out)
	}

	lines := bytes.Split(out, []byte{'\n'})
	c.Parents = make([]CommitID, len(lines)-2)
	for i, line := range lines {
		if i == 0 {
			// this commit is not its own parent, so skip it
			continue
		}
		if i == len(lines)-1 {
			// trailing newline means the last line is empty
			continue
		}
		c.Parents[i-1] = CommitID(line)
	}

	return c, nil
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

	// this just determines if the file exists.
	cmd := exec.Command("hg", "locate", "--rev="+string(fs.at), "--", path)
	cmd.Dir = fs.dir
	err := cmd.Run()
	if err != nil {
		// hg doesn't track dirs, so use a workaround to see if path is a dir.
		if _, err := fs.ReadDir(path); err == nil {
			return &fileInfo{name: filepath.Base(path), mode: os.ModeDir}, nil
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

	return &fileInfo{name: filepath.Base(path), size: int64(len(data))}, nil
}

func (fs *hgFSCmd) ReadDir(path string) ([]os.FileInfo, error) {
	path = filepath.Clean(path)
	cmd := exec.Command("hg", "locate", "--rev="+string(fs.at), "--include="+path, "--exclude="+filepath.Clean(path)+"/**/*")
	cmd.Dir = fs.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg cat` failed: %s. Output was:\n\n%s", err, out)
	}

	prefix := []byte(path + "/")
	files := bytes.Split(out, []byte{'\n'})
	var fis []os.FileInfo
	for _, nameb := range files {
		nameb = bytes.TrimPrefix(nameb, prefix)
		if len(nameb) == 0 {
			continue
		}
		if bytes.Contains(nameb, []byte{'/'}) {
			// subdir
			continue
		}
		// TODO(sqs): omits directories
		fis = append(fis, &fileInfo{name: filepath.Base(string(nameb))})
	}

	return fis, nil
}

func (fs *hgFSCmd) String() string {
	return fmt.Sprintf("hg repository %s commit %s (cmd)", fs.dir, fs.at)
}
