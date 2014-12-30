package vcs

// A Merger is a repository that can perform actions related to
// merging.
type Merger interface {
	// MergeBase returns the merge base commit for the specified
	// commits (aka greatest common ancestor commit for hg).
	MergeBase(CommitID, CommitID) (CommitID, error)
}
