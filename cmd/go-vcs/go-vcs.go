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
	"strconv"
	"time"

	"strings"

	"github.com/kr/text"
	vcs2 "github.com/shurcooL/go/vcs"
	"sourcegraph.com/sourcegraph/go-diff/diff"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	//_ "sourcegraph.com/sourcegraph/go-vcs/vcs/git"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/hg"
)

var (
	sshKeyFile = flag.String("i", "", "ssh key file")
	chdir      = flag.String("C", "", "change directory to this dir before doing anything")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("Must specify a subcommand.")
	}

	if *chdir != "" {
		if err := os.Chdir(*chdir); err != nil {
			log.Fatal("Chdir:", err)
		}
	}

	subcmd := flag.Arg(0)
	args := flag.Args()[1:]
	switch subcmd {
	case "git-clone-mirror":
		if len(args) != 2 {
			log.Fatal("git-clone requires 2 args: <clone URL> <dir>.")
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

		opt := vcs.CloneOpt{}
		if *sshKeyFile != "" {
			key, err := ioutil.ReadFile(*sshKeyFile)
			if err != nil {
				log.Fatal(err)
			}
			opt.RemoteOpts.SSH = &vcs.SSHConfig{PrivateKey: key}
		}
		repo, err := vcs.Clone("git", cloneURL.String(), dir, opt)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Cloned: %T.", repo)

	case "update-everything":
		if len(args) != 1 {
			log.Fatal("update-everything takes 1 argument: <repo dir>.")
		}
		dir := args[0]

		repo, err := vcs.Open("git", dir)
		if err != nil {
			log.Fatal(err)
		}

		getHEADCommit := func() *vcs.Commit {
			commitID, err := repo.ResolveRevision("HEAD")
			if err != nil {
				log.Fatal("Resolving HEAD revision:", err)
			}
			commit, err := repo.GetCommit(commitID)
			if err != nil {
				log.Fatal("GetCommit:", err)
			}
			return commit
		}

		preCommit := getHEADCommit()
		log.Printf("Before remote update, HEAD is %s (from %s ago).", preCommit.ID, preCommit.Author.Date)

		log.Printf("Remote-updating repo in dir %s...", dir)
		if err := repo.(vcs.RemoteUpdater).UpdateEverything(vcs.RemoteOpts{}); err != nil {
			log.Fatal(err)
		}

		postCommit := getHEADCommit()
		log.Printf("After remote update, HEAD is %s (from %s ago).", postCommit.ID, postCommit.Author.Date)

	case "show":
		if len(args) != 1 {
			log.Fatal("show takes 1 argument: <revspec>.")
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

	case "grep":
		if len(args) != 2 {
			log.Fatal("grep takes 2 arguments.")
		}

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		rev, err := repo.ResolveRevision(args[0])
		if err != nil {
			log.Fatal(err)
		}

		results, err := repo.(vcs.Searcher).Search(rev, vcs.SearchOptions{
			Query:        args[1],
			QueryType:    vcs.FixedQuery,
			ContextLines: 2,
			N:            5,
			Offset:       0,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("# %d matches", len(results))
		log.Println()
		for _, res := range results {
			log.Printf("# %s:%d-%d", res.File, res.StartLine, res.EndLine)
			fmt.Println(string(res.Match))
		}

	case "show-file":
		if len(args) != 2 {
			log.Fatal("show-file takes 2 arguments: <commit> <path>.")
		}

		repo, err := vcs.Open("git", ".")
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

	case "read-dir":
		if len(args) != 2 {
			log.Fatal("read-dir takes 2 arguments: <commit> <path>.")
		}

		started := time.Now()

		// Open using go/vcs to figure out VCS type (git, hg).
		r := vcs2.New(".")
		if r == nil {
			log.Fatalln("no supported vcs found in cwd")
		}

		repo, err := vcs.Open(r.Type().VcsType(), r.RootPath())
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

		startedReadDir := time.Now()
		fis, err := fs.ReadDir(args[1])
		readDirTaken := time.Since(startedReadDir)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# ReadDir (%d total):\n", len(fis))
		for _, fi := range fis {
			fmt.Println(fi.ModTime().Local(), fi.Name())
		}

		fmt.Println("fs.ReadDir() taken:", readDirTaken)
		fmt.Println("read-dir taken:", time.Since(started))

	case "log":
		if len(args) != 0 {
			log.Fatal("log takes no arguments.")
		}

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		commitID, err := repo.ResolveRevision("HEAD")
		if err != nil {
			log.Fatal(err)
		}

		commits, total, err := repo.Commits(vcs.CommitsOptions{Head: commitID, N: 250})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Commits (%d total):\n", total)
		for _, c := range commits {
			printCommit(c)
		}

	case "blame":
		newestCommit, file := args[0], args[1]

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		newestCommitID, err := repo.ResolveRevision(newestCommit)
		if err != nil {
			log.Fatal(err)
		}

		hunks, err := repo.(vcs.Blamer).BlameFile(file, &vcs.BlameOptions{NewestCommit: newestCommitID})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Hunks (%d total):\n", len(hunks))
		for _, c := range hunks {
			printHunk(c)
		}

	case "diff", "diffstat":
		if len(args) != 2 {
			log.Fatalf("%s takes 2 args (base and head), behavior is like `git diff base...head` (note triple dot).", subcmd)
		}
		baseRev := args[0]
		headRev := args[1]

		repo, err := vcs.Open("git", ".")
		if err != nil {
			log.Fatal(err)
		}

		base, err := repo.ResolveRevision(baseRev)
		if err != nil {
			log.Fatal(err)
		}
		head, err := repo.ResolveRevision(headRev)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("git diff %s..%s", base, head)

		vdiff, err := repo.(vcs.Differ).Diff(base, head, &vcs.DiffOptions{ExcludeReachableFromBoth: true})
		if err != nil {
			log.Fatal(err)
		}

		switch subcmd {
		case "diff":
			fmt.Println(vdiff.Raw)
		case "diffstat":
			pdiff, err := diff.ParseMultiFileDiff([]byte(vdiff.Raw))
			if err != nil {
				log.Fatal(err)
			}
			for _, fdiff := range pdiff {
				var name string
				if fdiff.NewName == "/dev/null" {
					name = fdiff.OrigName
				} else {
					name = fdiff.NewName
				}
				fmt.Printf("%-50s    ", name)
				st := fdiff.Stat()
				const w = 30
				total := st.Added + st.Changed + st.Deleted
				if st.Added > 0 {
					st.Added = (st.Added*w)/total + 1
				}
				if st.Changed > 0 {
					st.Changed = (st.Changed*w)/total + 1
				}
				if st.Deleted > 0 {
					st.Deleted = (st.Deleted*w)/total + 1
				}
				fmt.Print(strings.Repeat("+", int(st.Added)), strings.Repeat("Δ", int(st.Changed)), strings.Repeat("-", int(st.Deleted)), "\n")
			}
		}

	case "branches":
		if len(args) > 1 {
			log.Fatal("branches takes 0 or 1 arguments: [<behind-ahead-branch>].")
		}

		// Open using go/vcs to figure out VCS type (git, hg).
		r := vcs2.New(".")
		if r == nil {
			log.Fatalln("no supported vcs found in cwd")
		}

		repo, err := vcs.Open(r.Type().VcsType(), r.RootPath())
		if err != nil {
			log.Fatal(err)
		}

		var opt vcs.BranchesOptions
		if len(args) == 1 {
			opt.BehindAheadBranch = args[0]
		}
		branches, err := repo.Branches(opt)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Branches (%d total):\n", len(branches))
		for _, b := range branches {
			switch {
			case b.Counts == nil:
				fmt.Printf("%s %s\n", b.Head, b.Name)
			case b.Counts != nil:
				fmt.Printf("-%v | +%v | %s\n", b.Counts.Behind, b.Counts.Ahead, b.Name)
			}
		}

	case "history":
		if len(args) != 1 {
			log.Fatal("history takes 1 argument.")
		}
		path := args[0]

		// Open using go/vcs to figure out VCS type (git, hg).
		r := vcs2.New(".")
		if r == nil {
			log.Fatalln("no supported vcs found in cwd")
		}

		repo, err := vcs.Open(r.Type().VcsType(), r.RootPath())
		if err != nil {
			log.Fatal(err)
		}

		commitID, err := repo.ResolveRevision("HEAD")
		if err != nil {
			log.Fatal(err)
		}

		commits, total, err := repo.Commits(vcs.CommitsOptions{Head: commitID, N: 10, Path: path})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# History (%d total):\n", total)
		for _, c := range commits {
			printCommit(c)
		}

	case "committers":
		if len(args) > 2 {
			log.Fatal("committers takes at most 2 arguments.")
		}

		var opt vcs.CommittersOptions
		if len(args) > 0 {
			opt.Rev = args[0]
		}
		if len(args) > 1 {
			var err error
			opt.N, err = strconv.Atoi(args[1])
			if err != nil {
				log.Fatal(err)
			}
		}

		// Open using go/vcs to figure out VCS type (git, hg).
		r := vcs2.New(".")
		if r == nil {
			log.Fatalln("no supported vcs found in cwd")
		}

		repo, err := vcs.Open(r.Type().VcsType(), r.RootPath())
		if err != nil {
			log.Fatal(err)
		}

		committers, err := repo.Committers(opt)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Committers (%d total):\n", len(committers))
		for _, c := range committers {
			fmt.Printf("%s <%s> has %v commits\n", c.Name, c.Email, c.Commits)
		}

	case "file-lister":
		if len(args) != 1 {
			log.Fatal("history takes 1 argument: <commit>.")
		}

		// Open using go/vcs to figure out VCS type (git, hg).
		r := vcs2.New(".")
		if r == nil {
			log.Fatalln("no supported vcs found in cwd")
		}

		repo, err := vcs.Open(r.Type().VcsType(), r.RootPath())
		if err != nil {
			log.Fatal(err)
		}

		fileLister, ok := repo.(vcs.FileLister)
		if !ok {
			log.Println("repo is not a FileLister")
			break
		}

		rev, err := repo.ResolveRevision(args[0])
		if err != nil {
			log.Fatal(err)
		}

		files, err := fileLister.ListFiles(rev)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# Files (%d total):\n", len(files))
		for _, f := range files {
			fmt.Printf("%q\n", f)
		}
	}
}

func printCommit(c *vcs.Commit) {
	fmt.Printf("%s\n%s <%s> at %v\n%s\n\n", c.ID, c.Author.Name, c.Author.Email, c.Author.Date.Time(), text.Indent(c.Message, "\t"))
}

func printHunk(h *vcs.Hunk) {
	fmt.Printf("L%d-%d b%d-%d\t%s\t%v\n", h.StartLine, h.EndLine, h.StartByte, h.EndByte, h.CommitID, h.Author)
}
