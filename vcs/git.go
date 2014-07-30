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
	*GitRepositoryCmd
}

func (r *gitRepository) MirrorUpdate() error {
	return GitMirrorUpdate(r.dir)
}

// OpenGitRepository is an overwritable function that opens a git repository.
// The subpackage git_libgit2 overwrites this function (when imported) to use a
// libgit2-backed repository. Otherwise it uses GitRepositoryCmd.
var OpenGitRepository = func(dir string) (GitRepository, error) {
	return &gitRepository{dir, &GitRepositoryCmd{dir}}, nil
}

func GitMirrorUpdate(dir string) error {
	cmd := exec.Command("git", "remote", "update")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec `git remote update` failed: %s. Output was:\n\n%s", err, out)
	}
	return nil
}

func CloneGitRepository(urlStr, dir string) (GitRepository, error) {
	cmd := exec.Command("git", "clone", "--", urlStr, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `git clone` failed: %s. Output was:\n\n%s", err, out)
	}

	return OpenGitRepository(dir)
}
