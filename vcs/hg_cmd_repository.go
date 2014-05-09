package vcs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.google.com/p/go.tools/godoc/vfs"
)

type LocalHgCmdRepository struct {
	Dir string
}

func (r *LocalHgCmdRepository) ResolveRevision(spec string) (CommitID, error) {
	cmd := exec.Command("hg", "identify", "--debug", "-i", "--rev="+spec)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec `hg identify` failed: %s. Output was:\n\n%s", err, out)
	}
	return CommitID(bytes.TrimSpace(out)), nil
}

func (r *LocalHgCmdRepository) ResolveTag(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *LocalHgCmdRepository) FileSystem(at CommitID) (vfs.FileSystem, error) {
	return &localHgCmdFS{
		dir: r.Dir,
		at:  at,
	}, nil
}

type localHgCmdFS struct {
	dir string
	at  CommitID
}

func (fs *localHgCmdFS) Open(name string) (vfs.ReadSeekCloser, error) {
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

func (fs *localHgCmdFS) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *localHgCmdFS) Stat(path string) (os.FileInfo, error) {
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

func (fs *localHgCmdFS) ReadDir(path string) ([]os.FileInfo, error) {
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

func (fs *localHgCmdFS) String() string {
	return fmt.Sprintf("local hg cmd repository %s commit %s", fs.dir, fs.at)
}
