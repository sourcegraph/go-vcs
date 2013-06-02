package vcs_test

import (
	"fmt"
	"github.com/sqs/go-vcs"
	"io/ioutil"
	"os"
	"path/filepath"
)

func Example() {
	var tmpdir string
	tmpdir, err := ioutil.TempDir("", "go-vcs-TestGit")
	if err != nil {
		panic("TempDir: " + err.Error())
		return
	}
	defer os.RemoveAll(tmpdir)

	r, err := vcs.Git.Clone("https://bitbucket.org/sqs/go-vcs-gittest.git", tmpdir)
	if err != nil {
		fmt.Printf("Clone error: %s\n", err)
		return
	}

	// check out master
	masterDir, err := r.CheckOut("master")
	if err != nil {
		fmt.Printf("CheckOut master: %s\n", err)
		return
	}
	fmt.Printf("master foo: %s", readfile(filepath.Join(masterDir, "foo")))

	// check out a branch
	barbranchDir, err := r.CheckOut("barbranch")
	if err != nil {
		fmt.Printf("CheckOut barbranch: %s\n", err)
		return
	}
	fmt.Printf("barbranch bar: %s", readfile(filepath.Join(barbranchDir, "bar")))

	// check out a commit id
	barcommit := "f411e1ea59ed2b833291efa196e8dab80dbf7cb8"
	barcommitDir, err := r.CheckOut(barcommit)
	if err != nil {
		fmt.Printf("CheckOut barcommit %s: %s", barcommit, err)
		return
	}
	fmt.Printf("barcommit bar: %s", readfile(filepath.Join(barcommitDir, "bar")))

	// output:
	// master foo: Hello, foo
	// barbranch bar: Hello, bar
	// barcommit bar: Hello, bar
}

func readfile(path string) string {
	data, _ := ioutil.ReadFile(path)
	return string(data)
}
