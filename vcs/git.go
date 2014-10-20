package vcs

import (
	"fmt"
	"os/exec"
)

func init() {
	RegisterOpener("git", func(dir string) (Repository, error) {
		return OpenGitRepositoryCmd(dir)
	})
	RegisterCloner("git", func(url, dir string, opt CloneOpt) (Repository, error) {
		return CloneGitRepository(url, dir, opt)
	})
}

func OpenGitRepositoryCmd(dir string) (*GitRepositoryCmd, error) {
	return &GitRepositoryCmd{Dir: dir}, nil
}

func CloneGitRepository(url, dir string, opt CloneOpt) (*GitRepositoryCmd, error) {
	args := []string{"clone"}
	if opt.Bare {
		args = append(args, "--bare")
	}
	if opt.Mirror {
		args = append(args, "--mirror")
	}
	args = append(args, "--", url, dir)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git clone` failed: %s. Output was:\n\n%s", err, out)
	}
	return OpenGitRepositoryCmd(dir)
}
