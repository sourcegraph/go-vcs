package vcs

type GitRepository interface {
	Repository

	ResolveBranch(name string) (CommitID, error)
}
