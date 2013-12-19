package vcs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func jsonstr(o interface{}) string {
	s, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		panic(err.Error())
	}
	return string(s)
}

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

	wantCommits := []*Commit{
		{
			ID:          "bcc18e4692162e616cc6165589a24be4ea40e3d2",
			Message:     "bar",
			AuthorName:  "Quinn Slack",
			AuthorEmail: "qslack@qslack.com",
			AuthorDate:  mustParseJSONTime("2013-06-01T19:57:17-07:00"),
		},
		{
			ID:          "0c28a98a22ee21eaba25c78ef706f62b69f64527",
			Message:     "bar",
			AuthorName:  "Quinn Slack",
			AuthorEmail: "qslack@qslack.com",
			AuthorDate:  mustParseJSONTime("2013-06-01T19:40:15-07:00"),
		},
		{
			ID:          "d047adf8d7ff0d3c589fe1d1cd72e1b8fb9512ea",
			Message:     "foo",
			AuthorName:  "Quinn Slack",
			AuthorEmail: "qslack@qslack.com",
			AuthorDate:  mustParseJSONTime("2013-06-01T19:39:51-07:00"),
		},
	}
	commits, err := r.CommitLog()
	if err != nil {
		t.Fatalf("CommitLog: %s", err)
	}
	if !reflect.DeepEqual(wantCommits, commits) {
		t.Errorf("want commits == %v, got %v", jsonstr(wantCommits), jsonstr(commits))
	}

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

	c, err := r.CurrentCommitID()
	if err != nil {
		t.Fatalf("CurrentCommitID: %s", err)
	}
	if c != barcommit {
		t.Errorf("want CurrentCommitID == %q, got %q", barcommit, c)
	}

	if _, err := Clone(Hg, url, tmpdir); !os.IsExist(err) {
		t.Fatalf("Clone to existing dir: want os.IsExist(err), got %T %v", err, err)
	}

	// Open
	if r, err = Open(Hg, tmpdir); err != nil {
		t.Fatalf("Open: %s", err)
	}
	if defaultDir, err = r.CheckOut("default"); err != nil {
		t.Fatalf("CheckOut default: %s", err)
	}
	assertFileContains(t, defaultDir, "foo", "Hello, foo\n")
}
