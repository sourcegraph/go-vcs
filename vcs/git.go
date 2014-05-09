package vcs

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
