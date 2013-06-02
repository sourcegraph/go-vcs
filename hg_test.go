package vcs

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestHg(t *testing.T) {
	t.Parallel()

	var tmpdir string
	tmpdir, err := ioutil.TempDir("", "go-vcs-TestHg")
	if err != nil {
		t.Fatalf("TempDir: %s", err)
	}
	defer os.RemoveAll(tmpdir)

	url := "https://bitbucket.org/sqs/go-vcs-hgtest"
	r, err := Clone(Hg, url, tmpdir)
	if err != nil {
		t.Fatalf("Clone: %s", err)
	}

	// check out default
	defaultDir, err := r.CheckOut("default")
	if err != nil {
		t.Fatalf("CheckOut default: %s", err)
	}
	assertFileContains(t, defaultDir, "foo", "Hello, foo\n")
	assertNotFileExists(t, defaultDir, "bar")

	// check out a branch
	barbranchDir, err := r.CheckOut("barbranch")
	if err != nil {
		t.Fatalf("CheckOut barbranch: %s", err)
	}
	assertFileContains(t, barbranchDir, "bar", "Hello, bar\n")

	r.CheckOut("default")

	// check out a commit id
	barcommit := "bcc18e469216"
	barcommitDir, err := r.CheckOut(barcommit)
	if err != nil {
		t.Fatalf("CheckOut barcommit %s: %s", barcommit, err)
	}
	assertFileContains(t, barcommitDir, "bar", "Hello, bar\n")
}
