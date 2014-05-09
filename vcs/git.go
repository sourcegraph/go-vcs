package vcs

import (
	"fmt"
	"os/exec"
)

type GitRepository interface {
	Repository

	ResolveBranch(name string) (CommitID, error)
}

type gitRepository struct {
	dir string
	*GitRepositoryNative
	cmd *GitRepositoryCmd
}

func OpenGitRepository(dir string) (GitRepository, error) {
	native, err := OpenGitRepositoryNative(dir)
	if err != nil {
		return nil, err
	}

	return &gitRepository{dir, native, &GitRepositoryCmd{dir}}, nil
}

func CloneGitRepository(urlStr, dir string) (GitRepository, error) {
	cmd := exec.Command("git", "clone", "--", urlStr, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git clone` failed: %s. Output was:\n\n%s", err, out)
	}

	return OpenGitRepository(dir)
}
