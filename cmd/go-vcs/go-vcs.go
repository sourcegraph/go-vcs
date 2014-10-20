// The go-vcs program exposes go-vcs's library functionality through a
// command-line interface.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kr/text"
	"github.com/sourcegraph/go-vcs/vcs"
	_ "github.com/sourcegraph/go-vcs/vcs/git"
	_ "github.com/sourcegraph/go-vcs/vcs/hg"
)

var (
	sshKeyFile = flag.String("i", "", "ssh key file")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("Must specify a subcommand.")
	}

	subcmd := flag.Arg(0)
	args := flag.Args()[1:]
	switch subcmd {
	case "git-clone-mirror":
		if len(args) != 2 {
			log.Fatal("git-clone requires 2 args: clone URL and dir.")
		}
		cloneURLStr, dir := args[0], args[1]

		cloneURL, err := url.Parse(cloneURLStr)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			log.Fatalf("Clone destination dir must not exist: %s.", dir)
		}
		if _, err := os.Stat(filepath.Join(dir, "..")); err != nil {
			log.Fatalf("Clone destination dir parent must exist: %s.", filepath.Join(dir, ".."))
		}

		log.Printf("Cloning %s to %s...", cloneURL, dir)

		repo, err := vcs.Clone("git", cloneURL.String(), dir, vcs.CloneOpt{})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Cloned: %T.", repo)

	case "show":
		if len(args) != 1 {
			log.Fatal("show takes 1 argument (revspec).")
		}
		revspec := args[0]

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		commitID, err := repo.ResolveRevision(revspec)
		if err != nil {
			log.Fatal(err)
		}

		commit, err := repo.GetCommit(commitID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Revspec %q resolves to commit %s:\n", revspec, commitID)
		printCommit(commit)

	case "show-file":
		if err := os.Chdir("/tmp/vcsstore/hg/https/bitbucket.org/atlassian/connector-commons"); err != nil {
			log.Fatal(err)
		}

		if len(args) != 2 {
			log.Fatal("show-file takes 2 arguments.")
		}

		repo, err := vcs.Open("hg", ".")
		if err != nil {
			log.Fatal(err)
		}

		rev, err := repo.ResolveRevision(args[0])
		if err != nil {
			log.Fatal(err)
		}

		fs, err := repo.FileSystem(rev)
		if err != nil {
			log.Fatal(err)
		}

		f, err := fs.Open(args[1])
		if err != nil {
			log.Fatal(err)
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := os.Stdout.Write(b); err != nil {
			log.Fatal(err)
		}
	case "log":
		if len(args) != 0 {
			log.Fatal("log takes no arguments.")
		}

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		master, err := repo.ResolveRevision("master")
		if err != nil {
			log.Fatal(err)
		}

		commits, total, err := repo.Commits(vcs.CommitsOptions{Head: master})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Commits (%d total):\n", total)
		for _, c := range commits {
			printCommit(c)
		}

	case "branches":
		if len(args) != 0 {
			log.Fatal("branches takes no arguments.")
		}

		repo, err := vcs.Open("hg", ".")
		if err != nil {
			log.Fatal(err)
		}

		branches, err := repo.Branches()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Branches (%d total):\n", len(branches))
		for _, b := range branches {
			fmt.Printf("%s %s\n", b.Head, b.Name)
		}
	}
}

func printCommit(c *vcs.Commit) {
	fmt.Printf("%s\n%s <%s> at %s\n%s\n\n", c.ID, c.Author.Name, c.Author.Email, c.Author.Date, text.Indent(c.Message, "\t"))
}
