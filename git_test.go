package vcs

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func mustParseJSONTime(t string) time.Time {
	tm, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		panic(err.Error())
	}
	return tm
}

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

	// check commits (master only has 1 commit, so we run this check when we're
	// on barbranch, which has 2 commits)
	wantCommits := []*Commit{
		{
			ID:          "f411e1ea59ed2b833291efa196e8dab80dbf7cb8",
			Message:     "bar",
			AuthorName:  "Quinn Slack",
			AuthorEmail: "qslack@qslack.com",
			AuthorDate:  mustParseJSONTime("2013-06-01T11:53:24-07:00"),
		},
		{
			ID:          "d3dd4c84e9e429e28e05d53a04651bce084f0565",
			Message:     "foo",
			AuthorName:  "Quinn Slack",
			AuthorEmail: "qslack@qslack.com",
			AuthorDate:  mustParseJSONTime("2013-06-01T11:52:02-07:00"),
		},
	}
	commits, err := r.CommitLog()
	if err != nil {
		t.Fatalf("CommitLog: %s", err)
	}
	if !reflect.DeepEqual(wantCommits, commits) {
		t.Errorf("want commits == %v, got %v", jsonstr(wantCommits), jsonstr(commits))
	}

	// check out a commit id
	barcommit := "f411e1ea59ed2b833291efa196e8dab80dbf7cb8"
	barcommitDir, err := r.CheckOut(barcommit)
	if err != nil {
		t.Fatalf("CheckOut barcommit %s: %s", barcommit, err)
	}
	assertFileContains(t, barcommitDir, "bar", "Hello, bar\n")

	c, err := r.CurrentCommitID()
	if err != nil {
		t.Fatalf("CurrentCommitID: %s", err)
	}
	if c != barcommit {
		t.Errorf("want CurrentCommitID == %q, got %q", barcommit, c)
	}

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
