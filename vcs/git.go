package vcs

import (
	"fmt"
	"os/exec"
)

type GitRepository interface {
	Repository
}

type gitRepository struct {
	dir string
	*GitRepositoryLibGit2
	cmd *GitRepositoryCmd
}

func (r *gitRepository) MirrorUpdate() error {
	cmd := exec.Command("git", "remote", "update")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec `git remote update` failed: %s. Output was:\n\n%s", err, out)
	}
	return nil
}

func OpenGitRepository(dir string) (GitRepository, error) {
	native, err := OpenGitRepositoryLibGit2(dir)
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
