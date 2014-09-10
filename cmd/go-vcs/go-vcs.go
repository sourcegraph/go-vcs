// The go-vcs program exposes go-vcs's library functionality through a
// command-line interface.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/sourcegraph/go-vcs/vcs"
	_ "github.com/sourcegraph/go-vcs/vcs/git_libgit2"
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

		repo, err := vcs.CloneMirror("git", cloneURL.String(), dir)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Cloned: %T.", repo)

		// repo, err := vcs.Open("git", dir)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		master, err := repo.ResolveRevision("master")
		if err != nil {
			log.Fatal(err)
		}

		commits, total, err := repo.Commits(vcs.CommitsOptions{Head: master})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Commits (%d total):", total)
		for _, c := range commits {
			fmt.Printf("%s (%s <%s> at %s)\n", c.Message, c.Author.Name, c.Author.Email, c.Author.Date)
		}
	}
}
