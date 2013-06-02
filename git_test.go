package vcs

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGit(t *testing.T) {
	t.Parallel()

	var tmpdir string
	tmpdir, err := ioutil.TempDir("", "go-vcs-TestGit")
	if err != nil {
		t.Fatalf("TempDir: %s", err)
	}
	defer os.RemoveAll(tmpdir)

	url := "https://bitbucket.org/sqs/go-vcs-gittest.git"
	r, err := Clone(Git, url, tmpdir)
	if err != nil {
		t.Fatalf("Clone: %s", err)
	}

	// check out master
	masterDir, err := r.CheckOut("master")
	if err != nil {
		t.Fatalf("CheckOut master: %s", err)
	}
	assertFileContains(t, masterDir, "foo", "Hello, foo\n")
	assertNotFileExists(t, masterDir, "bar")

	// check out a branch
	barbranchDir, err := r.CheckOut("barbranch")
	if err != nil {
		t.Fatalf("CheckOut barbranch: %s", err)
	}
	assertFileContains(t, barbranchDir, "bar", "Hello, bar\n")

	// check out a commit id
	barcommit := "f411e1ea59ed2b833291efa196e8dab80dbf7cb8"
	barcommitDir, err := r.CheckOut(barcommit)
	if err != nil {
		t.Fatalf("CheckOut barcommit %s: %s", barcommit, err)
	}
	assertFileContains(t, barcommitDir, "bar", "Hello, bar\n")

	if _, err := Clone(Git, url, tmpdir); !os.IsExist(err) {
		t.Fatalf("Clone to existing dir: want os.IsExist(err), got %T %v", err, err)
	}

	// Open
	if r, err = Open(Git, tmpdir); err != nil {
		t.Fatalf("Open: %s", err)
	}
	if masterDir, err = r.CheckOut("master"); err != nil {
		t.Fatalf("CheckOut master: %s", err)
	}
	assertFileContains(t, masterDir, "foo", "Hello, foo\n")
}
