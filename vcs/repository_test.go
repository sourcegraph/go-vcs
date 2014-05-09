package vcs

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepository_ResolveRevision(t *testing.T) {
	defer removeTmpDirs()

	hgCommands := []string{
		"touch --date=2006-01-02T15:04:05Z f",
		"hg add f",
		"hg commit -m foo --date '2006-12-06 13:18:29 UTC' --user 'a <a@a.com>'",
	}
	tests := map[string]struct {
		repo         Repository
		spec         string
		wantCommitID CommitID
	}{
		"git": {
			repo: makeLocalGitRepository(t,
				"GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
			),
			spec:         "master",
			wantCommitID: "c556aa409427eed1322744a02ad23066f51040fb",
		},
		"hg": {
			repo:         makeLocalHgRepository(t, false, hgCommands...),
			spec:         "tip",
			wantCommitID: "e8e11ff1be92a7be71b9b5cdb4cc674b7dc9facf",
		},
		"hg cmd": {
			repo:         makeLocalHgRepository(t, true, hgCommands...),
			spec:         "tip",
			wantCommitID: "e8e11ff1be92a7be71b9b5cdb4cc674b7dc9facf",
		},
	}

	for label, test := range tests {
		commitID, err := test.repo.ResolveRevision(test.spec)
		if err != nil {
			t.Errorf("%s: ResolveRevision: %s", label, err)
			continue
		}

		if commitID != test.wantCommitID {
			t.Errorf("%s: got commitID == %v, want %v", label, commitID, test.wantCommitID)
		}
	}
}

func TestRepository_ResolveTag(t *testing.T) {
	defer removeTmpDirs()

	hgCommands := []string{
		"touch --date=2006-01-02T15:04:05Z f",
		"hg add f",
		"hg commit -m foo --date '2006-12-06 13:18:29 UTC' --user 'a <a@a.com>'",
		"hg tag t",
	}
	tests := map[string]struct {
		repo         Repository
		tag          string
		wantCommitID CommitID
	}{
		"git": {
			repo: makeLocalGitRepository(t,
				"GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
				"git tag t",
			),
			tag:          "t",
			wantCommitID: "c556aa409427eed1322744a02ad23066f51040fb",
		},
		"hg": {
			repo:         makeLocalHgRepository(t, false, hgCommands...),
			tag:          "t",
			wantCommitID: "e8e11ff1be92a7be71b9b5cdb4cc674b7dc9facf",
		},
		"hg cmd": {
			repo:         makeLocalHgRepository(t, true, hgCommands...),
			tag:          "t",
			wantCommitID: "e8e11ff1be92a7be71b9b5cdb4cc674b7dc9facf",
		},
	}

	for label, test := range tests {
		commitID, err := test.repo.ResolveTag(test.tag)
		if err != nil {
			t.Errorf("%s: ResolveTag: %s", label, err)
			continue
		}

		if commitID != test.wantCommitID {
			t.Errorf("%s: got commitID == %v, want %v", label, commitID, test.wantCommitID)
		}
	}
}

func TestRepository_FileSystem(t *testing.T) {
	defer removeTmpDirs()

	// In all tests, repo should contain two commits. The first commit (whose ID
	// is in the 'first' field) has a file at dir1/file1 with the contents
	// "myfile1" and the mtime 2006-01-02T15:04:05Z. The second commit (whose ID
	// is in the 'second' field) adds a file at file2 (in the top-level
	// directory of the repository) with the contents "infile2" and the mtime
	// 2014-05-06T19:20:21Z.
	//
	// TODO(sqs): add symlinks, etc.
	hgCommands := []string{
		"mkdir dir1",
		"echo -n infile1 > dir1/file1",
		"touch --date=2006-01-02T15:04:05Z dir1 dir1/file1",
		"hg add dir1/file1",
		"hg commit -m commit1 --user 'a <a@a.com>' --date '2006-01-02 15:04:05 UTC'",
		"echo -n infile2 > file2",
		"touch --date=2014-05-06T19:20:21Z file2",
		"hg add file2",
		"hg commit -m commit2 --user 'a <a@a.com>' --date '2014-05-06 19:20:21 UTC'",
	}
	tests := map[string]struct {
		repo          Repository
		first, second CommitID
	}{
		"git": {
			repo: makeLocalGitRepository(t,
				"mkdir dir1",
				"echo -n infile1 > dir1/file1",
				"touch --date=2006-01-02T15:04:05Z dir1 dir1/file1",
				"git add dir1/file1",
				"GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m commit1 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
				"echo -n infile2 > file2",
				"touch --date=2014-05-06T19:20:21Z file2",
				"git add file2",
				"GIT_COMMITTER_DATE=2014-05-06T19:20:21Z git commit -m commit2 --author='a <a@a.com>' --date 2014-05-06T19:20:21Z",
			),
			first:  "b57e3b5de36984ead5127a27f190fd69acb37fa4",
			second: "7c374610b4e4968b182ddfe2c220d033e62f0a3a",
		},
		"hg": {
			repo:   makeLocalHgRepository(t, false, hgCommands...),
			first:  "0b3260387c55ff0834b520fd7f5d4f4a15c22827",
			second: "810c55b76823441dabb1249837e7ebceab50ce1a",
		},
		"hg cmd": {
			repo:   makeLocalHgRepository(t, true, hgCommands...),
			first:  "0b3260387c55ff0834b520fd7f5d4f4a15c22827",
			second: "810c55b76823441dabb1249837e7ebceab50ce1a",
		},
	}

	for label, test := range tests {
		fs1, err := test.repo.FileSystem(test.first)
		if err != nil {
			t.Errorf("%s: FileSystem: %s", label, err)
			continue
		}

		// dir1 should exist and be a dir.
		dir1Info, err := fs1.Stat("dir1")
		if err != nil {
			t.Errorf("%s: fs1.Stat(dir1): %s", label, err)
			continue
		}
		if !dir1Info.Mode().IsDir() {
			t.Errorf("%s: dir1 stat !IsDir", label)
		}
		if name := dir1Info.Name(); name != "dir1" {
			t.Errorf("%s: got dir1 name %q, want 'dir1'", label, name)
		}

		// dir1 should contain one entry: file1.
		dir1Entries, err := fs1.ReadDir("dir1")
		if err != nil {
			t.Errorf("%s: fs1.ReadDir(dir1): %s", label, err)
			continue
		}
		if len(dir1Entries) != 1 {
			t.Errorf("%s: got %d dir1 entries, want 1", label, len(dir1Entries))
			continue
		}
		if file1Info := dir1Entries[0]; file1Info.Name() != "file1" {
			t.Errorf("%s: got dir1 entry name == %q, want 'file1'", label, file1Info.Name())
		}

		// dir1/file1 should exist, contain "infile1", and be a file.
		file1, err := fs1.Open("dir1/file1")
		if err != nil {
			t.Errorf("%s: fs1.Open(dir1/file1): %s", label, err)
			continue
		}
		file1Data, err := ioutil.ReadAll(file1)
		if err != nil {
			t.Errorf("%s: ReadAll(file1): %s", label, err)
			continue
		}
		if !bytes.Equal(file1Data, []byte("infile1")) {
			t.Errorf("%s: got file1Data == %q, want %q", label, string(file1Data), "infile1")
		}
		file1Info, err := fs1.Stat("dir1/file1")
		if err != nil {
			t.Errorf("%s: fs1.Stat(dir1/file1): %s", label, err)
			continue
		}
		if !file1Info.Mode().IsRegular() {
			t.Errorf("%s: file1 stat !IsRegular", label)
		}
		if name := file1Info.Name(); name != "file1" {
			t.Errorf("%s: got file1 name %q, want 'file1'", label, name)
		}
		if size, wantSize := file1Info.Size(), int64(len("infile1")); size != wantSize {
			t.Errorf("%s: got file1 size %d, want %d", label, size, wantSize)
		}

		// file2 shouldn't exist in the 1st commit.
		_, err = fs1.Open("file2")
		if !os.IsNotExist(err) {
			t.Errorf("%s: fs1.Open(file2): got err %v, want os.IsNotExist (file2 should not exist in this commit)", label, err)
		}

		// file2 should exist in the 2nd commit.
		fs2, err := test.repo.FileSystem(test.second)
		if err != nil {
			t.Errorf("%s: FileSystem: %s", label, err)
		}
		_, err = fs2.Open("file2")
		if err != nil {
			t.Errorf("%s: fs2.Open(file2): %s", label, err)
			continue
		}

		// file1 should also exist in the 2nd commit.
		file1Info, err = fs2.Stat("dir1/file1")
		if err != nil {
			t.Errorf("%s: fs2.Stat(dir1/file1): %s", label, err)
			continue
		}
		file1, err = fs2.Open("dir1/file1")
		if err != nil {
			t.Errorf("%s: fs2.Open(dir1/file1): %s", label, err)
			continue
		}
	}
}

var (
	tmpDirs     []string
	keepTmpDirs = flag.Bool("test.keeptmp", false, "don't remove temporary dirs after use")
)

func removeTmpDirs() {
	if *keepTmpDirs {
		return
	}
	for _, dir := range tmpDirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Fatalf("tearDown: RemoveAll(%q) failed: %s", dir, err)
		}
	}
	tmpDirs = nil
}

// makeTmpDir creates a temporary directory and returns its path. The directory
// is added to the list of directories to be removed when the currently running
// test ends (assuming the test calls removeTmpDirs() after execution).
func makeTmpDir(t testing.TB, suffix string) string {
	dir, err := ioutil.TempDir("", "go-vcs-"+suffix)
	if err != nil {
		t.Fatal(err)
	}

	if *keepTmpDirs {
		t.Logf("Using temp dir %s.", dir)
	}

	tmpDirs = append(tmpDirs, dir)
	return dir
}

func makeLocalGitRepository(t testing.TB, cmds ...string) GitRepository {
	dir := makeTmpDir(t, "git")
	cmds = append([]string{"git init"}, cmds...)
	for _, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = dir
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("Command %q failed. Output was:\n\n%s", cmd, out)
		}
	}

	r, err := OpenLocalGitRepository(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatal("OpenLocalGitRepository(%q) failed: %s", dir, err)
	}
	return r
}

func makeLocalHgRepository(t testing.TB, cmd bool, cmds ...string) Repository {
	dir := makeTmpDir(t, "hg")
	cmds = append([]string{"hg init"}, cmds...)
	for _, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = dir
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("Command %q failed. Output was:\n\n%s", cmd, out)
		}
	}

	if cmd {
		return &LocalHgCmdRepository{dir}
	}

	r, err := OpenLocalHgRepository(dir)
	if err != nil {
		t.Fatal("OpenLocalHgRepository(%q) failed: %s", dir, err)
	}
	return r
}
